yearly-summary
==============

Summarizes a GitHub user's commits for a year

Installation
============
You'll need a Go environment. Then just run `go build` or `go install`. You also need write access to a MongoDB server.

Usage
=====
Once the app is build, just run the executable (`go build` will generate the binary in the same directory as the source).
Use the `--help` flag to see the options. You will have to have a 
[GitHub Personal Access Token](https://github.com/blog/1509-personal-api-tokens) which must have access to the organization
and repositories you want to scan.

The required parameters are:
`--username` - The username of the GitHub user you want to generate the report for.
`--githubtoken` - The token discussed above
`--org` - The name of the organization on GitHub whose repositories you want to scan.

The application will iterate over all of the organization's repositories and get a list of all commits for the user over the
span of a year. When it is finished, it will produce a report which will have a line per repository similar to 
``repository_name: X.XX days``. 

The (big) assumption made is that any number of commits to a repository represent a day spent on that code. If commits are
made to multiple repositories, then the day is evenly split among them. For instance, if the user has 2 commits to ProjectA
and 1 commit to ProjectB on a given day, then both ProjectA and ProjectB earn half (0.5) a day. No weight is given to the 
number of commits or their size.
