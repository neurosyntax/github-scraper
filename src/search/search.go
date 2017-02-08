/*
    search.go

    A tool for scraping repositories from GitHub and extracting source code function information using Golang and GitHub API v3.
    
    Author: Justin Chen
    12.19.2016

    Boston University 
    Computer Science

    Dependencies:        exuberant ctags, and mongodb driver for go (http://labix.org/mgo)
    Operating systems:   GNU Linux, OS X
    Supported languages: C, C++, C#, Erlang, Lisp, Lua, Java, Javascript, and Python
*/

package search

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
    URL string `json:"url"`*/
    CreatedAt string `json:"created_at"`
    /*UpdatedAt string `json:"updated_at"`
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

type Document interface {
    getObjectIdStr() string
}

type RepoDoc struct {
    Id         uint32 `json:"id" bson:"_id,omitempty"`
    Owner      string
    RepoName   string
    RepoURL    string
    RepoSizeKB int
    RepoLang   string
    FilePath   string
}

func (r RepoDoc) getObjectIdStr() string {
    return strings.Split(r.Id.String(), "\"")[1]
}

type FuncDoc struct {
    Id bson.ObjectId `json:"id" bson:"_id,omitempty"`
    RepoId string
    RawURL string
    FileName string
    FuncName string
    InputType string
    OutputType string
    ASTID string
    CFGID string
}

func (f FuncDoc) getObjectIdStr() string {
    return strings.Split(f.Id.String(), "\"")[1]
}

func getWaitTime() time.Duration {
    curl := exec.Command("curl", "-i", "https://api.github.com/")
    grep  := exec.Command("grep", "X-RateLimit-Reset:")
    awk   := exec.Command("awk", "{$1=\"\"; print $0}")
    grep.Stdin, _ = curl.StdoutPipe()
    awk.Stdin, _  = grep.StdoutPipe()
    awkOut, _    := awk.StdoutPipe()
    buff := bufio.NewScanner(awkOut)
    var header []string

    _ = grep.Start()
    _ = awk.Start()
    _ = curl.Run()
    _ = grep.Wait()
    defer awk.Wait()

    for buff.Scan() {    
        header = append(header, buff.Text()+"\n")
    }

    utcInt64, _ := strconv.ParseInt(strings.TrimSpace(header[0]), 10, 64)

    return time.Unix(utcInt64, 0).Sub(time.Now())
}

func search(query string, queryResp interface{}, username string, password string) bool {
    client := &http.Client{}
    req, err := http.NewRequest("GET", query, nil)
    
    if strings.Compare(username, "") != 0 && strings.Compare(password, "") != 0 {
        req.SetBasicAuth(username, password)
    }
    
    resp, err := client.Do(req)
    
    check(err)
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)

    check(err)
    err = json.Unmarshal(body, &queryResp)
    
    if err != nil {
        var errorResp NotFoundResp
        json.Unmarshal(body, &errorResp)

        return !strings.Contains(errorResp.Message, "API rate limit exceeded")
    }

    return true
}