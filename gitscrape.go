/*
	gitscrape.go
	
	Author: Justin Chen
	12.19.2016

	Boston University 
	Computer Science
*/

package main

import (
	"fmt"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"encoding/json"
	"os"
    "os/exec"
	"strings"
	"bytes"
    "bufio"
    "sync"
)

type githubSearchObj struct {
	TotalCount int `json:"total_count"`
	Items []searchItems `json:"items"`
}

type searchItems struct {
	Name string `json:"name"`
	CloneURL string `json:"clone_url"`
}

func main() {

    // Parameters found here: https://developer.github.com/v3/search/
    fmt.Print("Github search scraper\nSearch options [-q, -i, -size, -forks, -forked, -created, -updated, -user, -repo, -lang, -stars -sort -order]\n")
    flag.String("q", "", "The search terms and any qualifiers.")
    flag.String("in", "", "Qualifies which fields are searched. With this qualifier you can restrict the search to just the repository name, description, readme, or any combination of these.")
    flag.Int("size", 1, "Finds repositories that match a certain size (in kilobytes).")
    flag.Int("forks", 0, "Filters repositories based on the number of forks.")
    flag.Bool("fork", false, "Filters whether forked repositories should be included (true) or only forked repositories should be returned (only).")
    flag.String("created", "", "Filters repositories based on date of creation.")
    flag.String("updated", "", "Filters repositories based on date they were last updated.")
    flag.String("user", "", "Limits searches to a specific user.")
    flag.String("repo", "", "Limits searches to a specific repository.")
    flag.String("language", "", "Searches repositories based on the programming language they're written in.")
    flag.Int("stars", 0, "Searches repositories based on the number of stars.")
    flag.String("sort", "", "The sort field. One of stars, forks, or updated. Default: results are sorted by best match.")
    flag.Bool("order", false, "The sort order if sort parameter is provided. One of asc (true) or desc (false). Default: false")
    flag.Parse()

    var query bytes.Buffer
    query.WriteString("https://api.github.com/search/repositories?")

    // grab all arguments and create the Github search query
    for i, arg := range os.Args {

    	if i > 0 {
    		if strings.HasPrefix(arg, "-") {
    			if strings.Compare("-q", arg) == 0 {
	    			query.WriteString("q=")
		    	} else if strings.Compare("-sort", arg) == 0 {
					query.WriteString("&sort=")
				} else if strings.Compare("-order", arg) == 0 {
					query.WriteString("&order=")
				} else {
					query.WriteString("+"+arg[1:]+":")
				}
     		} else {
                query.WriteString(arg)
            }
		}
    }

    reader := bufio.NewReader(os.Stdin)
    fmt.Print("directory to clone all repos to: ")
    dir, _ := reader.ReadString('\n')
    dir = strings.Replace(dir, "\n", "", -1)

    if !dirExists(dir) {
        os.Mkdir(dir, os.FileMode(0777))
    }
    var queryRespObj = search(query)
    log.Printf("%+v", queryRespObj)
    //massivelyClone(search(query), dir)
}

func search(query bytes.Buffer) githubSearchObj {
    resp, err := http.Get(query.String())
    
    if err != nil {
        log.Fatal(err)
    }

    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)

    if err != nil {
        log.Fatal(err)
    }

    var queryRespObj githubSearchObj
    err = json.Unmarshal(body, &queryRespObj)

    if err != nil {
        log.Fatal(err)
    }

    return queryRespObj
}

func dirExists(dir string)  bool {
    _, err := os.Stat(dir)
    if err == nil {
        return true
    }
    if os.IsNotExist(err) {
        return false
    }
    return true
}

func massivelyClone(queryRespObj githubSearchObj, dir string) {

    tasks := make(chan *exec.Cmd, 64)

    // spawn four worker goroutines
    var wg sync.WaitGroup
    for i := 0; i < 4; i++ {
        wg.Add(1)
        go func() {
            for cmd := range tasks {
                cmd.Run()
            }
            wg.Done()
        }()
    }
    
    os.Chdir(dir)
    for _, repo := range queryRespObj.Items {
        //cmd := exec.Command("git", "clone", repo.CloneURL)
        tasks <- exec.Command("git", "clone", repo.CloneURL)
        //err := cmd.Run()
        //if err != nil {
            // something went wrong
        //}
    }
    close(tasks)
    wg.Wait()
}