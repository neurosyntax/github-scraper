# github-scraper![Go gopher](./images/gopherbelly50.jpg)

Language: Go (Golang)

A tool for scraping repositories using Golang and GitHub API v3. Assumes you're running Linux or OSX.

#### Setup/ Dependencies

[Install MongoDB](https://golang.org/doc/install)

[Install MongoDB](https://docs.mongodb.com/manual/tutorial/install-mongodb-on-ubuntu/)

Install Exuberant Ctags:
```sh
Ubuntu:
sudo apt install exuberant-ctags

Unix:
[Download](http://ctags.sourceforge.net/)
```
Install MongoDB driver for Go:
```sh
go get gopkg.in/mgo.v2
```
Refer to [mgo](https://github.com/go-mgo/mgo) for further and more up-to-date instructions.

GOPATH setup
```sh
Linux:
sudo nano ~/.bashrc
export GOPATH=$HOME/<path to this repo>
source ~/.bashrc

Unix:
sudo nano ~/.bash_profile
export GOPATH=$HOME/<path to this repo>
touch ~/.bash_profile
```

If you get a message about GOPATH not able to find `parse/parseFuncHeader.go`, do the following:
```sh
cd github-scraper/src/parse
go build
go install
```

#### Basic usage:
```sh
go run gitscrape.go -q <search terms> -language <programming language>
username:
password:
```
This will search for all repositories that match the specified search terms. You will be prompted for your username and password. Github API's have a "rate limit", which only allows a public IP to make 60 request/hour if the request is not authenticated with a Github account. If the request is made with basic authentication or oauth, a client can make up to 5,000 request/hour. The `per_page` parameter is set to `100` items by default.

After authenticating, you will then be presented with a prompt for a directory name where all the repos will be cloned to.
```sh
directory to clone all repos to:
```
Github search API arguments you can give to the program.
Look here for more details: [Search code parameters](https://developer.github.com/v3/search/)
```sh
[-q, -i, -size, -forks, -forked, -created, -updated, -user, -repo, -lang, -stars -sort -order]
```
Github's API expects the flags to be ordered as in the list above for correct results.

Additional flags:
```sh
-all  Start search from April 10th, 2008 and use >=2008-04-10T00:00:00Z instead of -created flag.
-mc   Massively clone all repos returned by search. Haven't finished this feature yet.
```

#### Indefinite usage:
```sh
go run gitscrape.go -q <search terms> -language <programming language> -all
authenticate [y/n]: y
directory to clone all repos to: saved_repos
use most recent checkpoint [y/n]: n
```

Note:
The `-all` flag is meant to try to circumvent the rate limit by searching through time. GitHub's Search API rate limit only allows 1,000 results per unique search and each search must contain some search term specified with `-q` (a single whitespace counts). This means, for example, if you want to search for all repositories written in Java, you will only the first 1,000 from the search even though there are 2,701,409 (at the time of writting this). I could specify a reverse order and get another 1,000 repos from that search, however, I still feel cheated. To get as many repos from a search as possible, specifying `-all` starts the search from the launch date of GitHub and looks for repos when they were created with the `-created` flag and makes the assumption that Java repos are created at a rate of about 1,000 Java repos/6 hours (rough estimate from observing the search api for one minute). Also from observing the difference in total_count from `https://api.github.com/search/repositories?q=created:%3C=2016-04-21T00:00:00Z+language:java&per_page=100` and `https://api.github.com/search/repositories?q=created:%3C=2016-04-21T06:00:00Z+language:java&per_page=100`, you I observed 1808480-1807687 = 793 Java repos created in that 6 hours. Those dates were picked randomly. So query over 6 hours intervals seems appropriate. To be a bit more conservative, I cut that in half and only search by 3 hour intervals to get as many as possible. You can adjust assumption this as needed.