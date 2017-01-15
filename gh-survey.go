/*
	gitscrape.go
	
	Author: Justin Chen
	12.19.2016

	Boston University 
	Computer Science

    Dependencies:      exuberant ctags, and mongodb driver for go (http://labix.org/mgo)
    Operating systems: GNU Linux, OS X
*/

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"encoding/json"
	"strings"
    "strconv"
    "bytes"
)

// Github Search API response object
type GithubSearchResp struct {
	TotalCount int `json:"total_count"`
    // IsIncomplete bool `json:"incomplete_results"`
	Items []Searchitems `json:"items"`
}

type Searchitems struct {
    // Id int `json:"id"`
	Name string `json:"name"`
    // FullName string `json:"full_name"`
    Owner OwnerItem `json:"owner"`
    CloneURL string `json:"clone_url"`
/*    IsPrivate bool `json:"private"`
    HTMLURL string `json:"html_url"`
    Description string `json:"description"`
    IsFork bool `json:"fork"`
    URL string `json:"url"`
    CreatedAt string `json:"created_at"`
    UpdatedAt string `json:"updated_at"`
    PushedAt string `json:"pushed_at"`
    Homepage string `json:"homepage"`*/
    SizeKB int `json:"size"`
    /*Stargazers int `json:"stargazers_count"`
    Watchers int `json:"watchers_count"`*/
    Language string `json:"language"`
    /*Forks int `json:"forks_count"`
    OpenIssues int `json:"open_issues_count"`
    MasterBranch string `json:"master_branch"`
    DefaultBranch string `json:"default_branch"`
    Score float64 `json:"score"`*/
}

type OwnerItem struct {
    Login string `json:"login"`
/*    Id int `json:"id"`
    AvatarURL string `json:"avatar_url"`
    GravatarID string `json:"gravatar_id"`
    URL string `json:"url"`
    ReceivedEventsURL string `json:"received_events_url"`
    OwnerType string `json:"url"`*/
}

// Github content API response object
type GithubContentResp struct {
    ContentType string `json:"type"`
/*    Encoding string `json:"encoding"`
    Size int `json:"size"`*/
    Name string `json:"name"`
/*    Path string `json:"path"`
    Content string `json:"content"`
    SHA string `json:"sha"`
    URL string `json:"url"`
    GitURL string `json:"git_url"`
    HTMLURL string `json:"html_url"`*/
    DownloadURL string `json:"download_url"`
    // _Links ContentLinks `json:"_links"`
}

type NotFoundResp struct {
    Message string `json:"message"`
    DocURL string `json:"documentation_url"`
}

/*type ContentLinks struct {
    Git string `json:"git"`
    Self string `json:"self"`
    HTML string `json:"html"`
}*/

func main() {
    totalSearchSize := 0

    for page := 1; page <= 10; page++ {
        // Search query queue
        var searchResp GithubSearchResp
        var searchQuery bytes.Buffer
        searchQuery.WriteString("https://api.github.com/search/repositories?q=language:Java&page="+strconv.Itoa(page)+"per_page=100")
        search(searchQuery, &searchResp)

        for _, repo := range searchResp.Items {
            fmt.Println(repo.SizeKB)
            totalSearchSize += repo.SizeKB
        }
    }
    totalSearchSize /= 1000000
    fmt.Println("Total size (GB): "+strconv.Itoa(totalSearchSize))
}


func search(query bytes.Buffer, queryResp interface{}) bool {
    client := &http.Client{}
    req, _ := http.NewRequest("GET", query.String(), nil)
    
    resp, _ := client.Do(req)
    
    defer resp.Body.Close()
    body, _ := ioutil.ReadAll(resp.Body)

    err := json.Unmarshal(body, &queryResp)
    
    if err != nil {
        var errorResp NotFoundResp
        json.Unmarshal(body, &errorResp)

        return !strings.Contains(errorResp.Message, "API rate limit exceeded")
    }

    return true
}