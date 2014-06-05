package main

import (
	"code.google.com/p/goauth2/oauth"
	"flag"
	"fmt"
	"github.com/google/go-github/github"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"strconv"
	"time"
)

type RepoCommit struct {
	Repo    string     `bson:"repo"`
	Author  *string    `bson:"author"`
	Date    *time.Time `bson:"date"`
	Message *string    `bson:"message"`
	Sha     *string    `bson:"sha"`
}

type DayCount struct {
	Day   int      `bson:"_id"`
	Count int      `bson:"count"`
	Repos []string `bson:"repos"`
}

func main() {
	var githubUser string
	flag.StringVar(&githubUser, "username", "", "The GitHub username")
	mongoHost := flag.String("mongohost", "localhost", "MongoDB host")
	mongoPort := flag.Int("mongoport", 27017, "MongoDB port")
	mongoDatabase := flag.String("mongodatabase", "yearly-summary", "MongoDB database")
	githubToken := flag.String("githubtoken", "", "GitHub access token")
	organization := flag.String("org", "", "Name of organization on GitHub whose repos will be scanned")
	var year int
	flag.IntVar(&year, "year", 2013, "Year to summarize")

	flag.Parse()

	if githubUser == "" || *githubToken == "" || *organization == "" {
		fmt.Println("username, githubtoken, and org are all required")
		return
	}

	session, _ := mgo.Dial(fmt.Sprintf("mongodb://%v:%v", *mongoHost, *mongoPort))
	session.SetSafe(&mgo.Safe{W: 1})
	db := session.DB(*mongoDatabase)
	defer session.Close()
	collection := db.C(githubUser)

	transport := &oauth.Transport{
		Token: &oauth.Token{AccessToken: *githubToken},
	}

	errorCount := 0

	client := github.NewClient(transport.Client())
	commitsListOptions := &github.CommitsListOptions{
		Author: githubUser,
		Since:  time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC),
		Until:  time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC),
	}

	nextPage := 1
	for nextPage > 0 {
		opt := &github.RepositoryListByOrgOptions{}
		opt.Page = nextPage
		fmt.Println("Processing page", nextPage, "of repos")
		repos, response, err := client.Repositories.ListByOrg(*organization, opt)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		nextPage = response.NextPage

		for _, repo := range repos {
			repoName := *(repo.Name)
			fmt.Println("  Processing repo", repoName)
			commitsPage := 1
			for commitsPage > 0 {
				commitsListOptions.Page = commitsPage
				commits, response, err := client.Repositories.ListCommits(*organization, repoName, commitsListOptions)
				if err != nil {
					fmt.Println("Error:", err)
					errorCount += 1
					commitsPage = 0
					continue
				}
				fmt.Println("    Page", commitsPage, "-", len(commits), "commits")
				commitsPage = response.NextPage

				for _, commit := range commits {
					commitPart := commit.Commit
					author := commitPart.Author
					repoCommit := RepoCommit{repoName, author.Name, author.Date, commitPart.Message, commit.SHA}
					collection.Insert(repoCommit)
				}
			}
		}
	}

	pipe := collection.Pipe([]bson.M{
		{"$group": bson.M{
			"_id":   bson.M{"$dayOfYear": "$date"},
			"count": bson.M{"$sum": 1},
			"repos": bson.M{"$addToSet": "$repo"},
		}},
		{"$out": githubUser + "_reduced"},
	})
	iterator := pipe.Iter()
	iterator.Close()

	reducedCollection := db.C(githubUser + "_reduced")
	var dayCounts []DayCount
	reducedCollection.Find(bson.M{}).Limit(365).All(&dayCounts)

	longestRepo := 0
	repoDays := make(map[string]float64)
	for _, dayCount := range dayCounts {
		days := 1 / float64(len(dayCount.Repos))
		for _, repo := range dayCount.Repos {
			if len(repo) > longestRepo {
				longestRepo = len(repo)
			}
			repoDays[repo] += days
		}
	}

	for repo, days := range repoDays {
		fmt.Printf("%"+strconv.Itoa(longestRepo)+"v: %6.2f days\n", repo, days)
	}
}
