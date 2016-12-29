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
	"strings"
	"bytes"
    "bufio"
    // "reflect"
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
/*    IsPrivate bool `json:"private"`
    HTMLURL string `json:"html_url"`
    Description string `json:"description"`
    IsFork bool `json:"fork"`
    URL string `json:"url"`
    CreatedAt string `json:"created_at"`
    UpdatedAt string `json:"updated_at"`
    PushedAt string `json:"pushed_at"`
    Homepage string `json:"homepage"`
    Size int `json:"size"`
    Stargazers int `json:"stargazers_count"`
    Watchers int `json:"watchers_count"`
    Language string `json:"language"`
    Forks int `json:"forks_count"`
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
    HTMLURL string `json:"html_url"`
    DownloadURL string `json:"download_url"`
    _Links ContentLinks `json:"_links"`*/
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
    // Choose directory to save repos to
    reader := bufio.NewReader(os.Stdin)
    fmt.Print("username: ")
    un, _ := reader.ReadString('\n')
    un = strings.Replace(un, "\n", "", -1)
    fmt.Print("password:")
    pw, _ := reader.ReadString('\n')
    pw = strings.Replace(pw, "\n", "", -1)

    /*
    fmt.Print("directory to clone all repos to: ")
    dir, _ := reader.ReadString('\n')
    dir = strings.Replace(dir, "\n", "", -1)

    // Make directory if it does not already exist
    if !dirExists(dir) {
        os.Mkdir(dir, os.FileMode(0777))
    }*/

    // Parameters found here: https://developer.github.com/v3/search/
    flag.String("u", "", "Authenticate with username.")
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

    var searchQuery bytes.Buffer
    searchQuery.WriteString("https://api.github.com/search/repositories?")

    // Grab all arguments and create the Github search query
    for i, arg := range os.Args {
        arg = strings.ToLower(arg)

    	if i > 0 {
    		if strings.HasPrefix(arg, "-") {
    			if strings.Compare("-q", arg) == 0 {
	    			searchQuery.WriteString("q=")
		    	} else if strings.Compare("-sort", arg) == 0 {
					searchQuery.WriteString("&sort=")
				} else if strings.Compare("-order", arg) == 0 {
					searchQuery.WriteString("&order=")
				} else if strings.Compare("-u", arg) == 0 {
                    searchQuery.WriteString("&u=")
                } else {
					searchQuery.WriteString("+"+arg[1:]+":")
				}
     		} else {
                searchQuery.WriteString(arg)
            }
		}
    }

    // Search query
    var searchResp GithubSearchResp
    search(searchQuery, &searchResp, un, pw)
    log.Printf("%+v\n", searchResp)


    for _, repo := range searchResp.Items {

        // url for this particular repo
        var contentQuery bytes.Buffer
        contentQuery.WriteString("https://api.github.com/repos/")
        contentQuery.WriteString(strings.Join([]string{repo.Owner.Login, repo.Name, "contents"}, "/")) //append GET /repos/:owner/:repo/contents/:path, but :path is optional 
        
        // Get the contents in the home directory of this repo
        fmt.Println("next repo")
        var contentResp []GithubContentResp
        search(contentQuery, &contentResp, un, pw)

        // BFS on this repo
        for (0 < len(contentResp)) {

            // Dequeue
            cont       := contentResp[0]
            contentResp = contentResp[1:]

            if strings.Compare(cont.ContentType, "file") == 0 && strings.HasSuffix(cont.Name, ".java") {
                // Parse file and search for functions of desired type
            } else if strings.Compare(cont.ContentType, "dir") == 0 {
                // Construct the url to search this sub-directory
                var contentDir bytes.Buffer
                contentDir.WriteString("https://api.github.com/repos/")
                contentDir.WriteString(strings.Join([]string{repo.Owner.Login, repo.Name, "contents", cont.Name}, "/"))
                fmt.Println(contentDir.String())
                var subdirContentResp []GithubContentResp
                search(contentDir, &subdirContentResp, un, pw)

                for  _, subdirCont := range subdirContentResp {
                    // Enqueue contents of sub-directory
                    if strings.Compare(subdirCont.ContentType, "file") == 0 && strings.HasSuffix(subdirCont.Name, ".java") {
                        contentResp = append(contentResp, subdirCont)
                    } else if strings.Compare(subdirCont.ContentType, "dir") == 0 {
                        // If a directory, need to prepend current dir's name to its subdir's name so it can search correctly
                        // in the outter loop above
                        subdirCont.Name = strings.Join([]string{cont.Name, subdirCont.Name}, "/")
                        contentResp = append(contentResp, subdirCont)
                    }
                }
            }
        }
        // log.Printf("%+v", contentResp)
    }
}

func search(query bytes.Buffer, queryResp interface{}, username string, password string) {

    client := &http.Client{}
    req, err := http.NewRequest("GET", query.String(), nil)
    req.SetBasicAuth(username, password)
    resp, err := client.Do(req)
    // resp, err := http.Get(query.String())
    
    if err != nil {
        log.Fatal(err)
    }

    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)

    if err != nil {
        log.Fatal(err)
    }
    // fmt.Println(len(body))
    err = json.Unmarshal(body, &queryResp)
    
    if err != nil {
        var errorResp NotFoundResp
        var badResp = json.Unmarshal(body, &errorResp)

        if badResp == nil {
            log.Printf("error: %+v\n", errorResp)
            log.Fatal()
        }
    }
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


/*
func readLines(path string) ([]string, error) {
  file, err := os.Open(path)
  if err != nil {
    return nil, err
  }
  defer file.Close()

  var lines []string
  scanner := bufio.NewScanner(file)
  for scanner.Scan() {
    lines = append(lines, scanner.Text())
  }
  return lines, scanner.Err()
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

*/