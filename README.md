# github-scraper

![Go gopher](./images/gopherbelly50.jpg)

Language: Go (Golang) 

#### Basic usage:
```sh
$ go run gitscrape.go -q <search terms>
```
This will search for all repositories that match the specified search terms. You will then be presented with a prompt for a directory name where all the repos will be cloned to.
```sh
$ directory to clone all repos to:
```
Github search API arguments you can give to the program.
Look here for more details: [Search code parameters](https://developer.github.com/v3/search/)
```sh
[-q, -i, -size, -forks, -forked, -created, -updated, -user, -repo, -lang, -stars -sort -order]
```