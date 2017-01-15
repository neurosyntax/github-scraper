# github-scraper

![Go gopher](./images/gopherbelly50.jpg)

Language: Go (Golang) 

A tool for scraping repositories using Golang and GitHub API v3.

#### Setup
Assumes you're running Linux or OSX, and have Golang and MongoDB installed.
Install Exuberant Ctags:
```sh
$ sudo apt install exuberant-ctags
```
Install MongoDB driver for Go:
```sh
$ go get gopkg.in/mgo.v2
```
Refer to [mgo](https://github.com/go-mgo/mgo) for further and more up-to-date instructions.

GOPATH setup
```sh
export GOPATH=$HOME/<path to this repo>
```

#### Basic usage:
```sh
$ go run gitscrape.go -q <search terms> -language <programming language>
$ username:
$ password:
```
This will search for all repositories that match the specified search terms. You will be prompted for your username and password. Github API's have a "rate limit", which only allows a public IP to make 60 request/hour if the request is not authenticated with a Github account. If the request is made with basic authentication or oauth, a client can make up to 5,000 request/hour.

After authenticating, you will then be presented with a prompt for a directory name where all the repos will be cloned to.
```sh
$ directory to clone all repos to:
```
Github search API arguments you can give to the program.
Look here for more details: [Search code parameters](https://developer.github.com/v3/search/)
```sh
[-q, -i, -size, -forks, -forked, -created, -updated, -user, -repo, -lang, -stars -sort -order]
```
Github's API expects the flags to be ordered as in the list above for correct results.