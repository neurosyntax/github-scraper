/*
	gitscrape.go

    A tool for scraping repositories from GitHub and extracting source code function information using Golang and GitHub API v3.
	
	Author: Justin Chen
	12.19.2016

	Boston University 
	Computer Science

    Dependencies:        exuberant ctags, and mongodb driver for go (http://labix.org/mgo)
    Operating systems:   GNU Linux, OS X
    Supported languages: C, C++, C#, Erlang, Lisp, Lua, Java, Javascript, and Python
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
    "strconv"
	"bytes"
    "bufio"
    "sync"
    "runtime"
    "time"
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
    "parse"
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
    Id         bson.ObjectId `json:"id" bson:"_id,omitempty"`
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

type FuncAST struct {
    Id bson.ObjectId `json:"id" bson:"_id,omitempty"`
    FuncID string
    FuncName string
    SavePath string
}

func (ast FuncAST) getObjectIdStr() string {
    return strings.Split(ast.Id.String(), "\"")[1]
}

type FuncCFG struct {
    Id bson.ObjectId `json:"id" bson:"_id,omitempty"`
    FuncID string
    FuncName string
    SavePath string
}

func (cfg FuncCFG) getObjectIdStr() string {
    return strings.Split(cfg.Id.String(), "\"")[1]
}

// ISO8601 date format
type Date struct {
    CompareOp string
    UTC time.Time
}

func (d Date) Increment(year int, month int, day int, hour int, min int, sec int) time.Time {
    dur, _ := time.ParseDuration(strconv.Itoa(hour)+"h"+strconv.Itoa(min)+"m"+strconv.Itoa(sec)+"s")
    fmt.Println(dur.String())
    return d.UTC.AddDate(year, month, day).Add(dur)
}

func (d Date) String() string {
    date := strings.Split(d.UTC.String(), " ")
    return date[0]+"T"+date[1]+"Z"
}

func StrTimeDate(iso string) time.Time {
    st     := strings.Split(strings.Replace(iso, "Z", "", -1), "T")
    date   := strings.Split(st[0], "-")
    ti     := strings.Split(st[1], ":")
    y, _   := strconv.Atoi(date[0])
    mo, _  := strconv.Atoi(date[1])
    d, _   := strconv.Atoi(date[2])
    h, _   := strconv.Atoi(ti[0])
    min, _ := strconv.Atoi(ti[1])
    s, _   := strconv.Atoi(ti[2])
    return time.Date(y, time.Month(mo), d, h, min, s, 0, time.UTC)
}

func GetTime(year int, month int, day int, hour int, min int, sec int) time.Time {
    if year <= 2007 {
        log.Printf("warning: year: %d is invalid. GitHub's API does not search back pass 2007", year)
    }
    return time.Date(year, time.Month(month), day, hour, min, sec, 0, time.UTC)
}

func getLangExt (lang string) string {
    langMap := map[string]string {"c":"c", "c++":"cpp", "cpp":"cpp", "c#":"cs",
                                  "cs":"cs", "erlang":"erl", "java":"java",
                                  "javascript":"js", "lisp":"lsp", "lua":"lua", "python":"py"}
    return langMap[strings.TrimSpace(lang)]
}

func getFuncTerm (ext string) string {
    extMap := map[string]string {"c":"function", "cpp":"function", "cs":"method",
                                 "erl":"function", "java":"method", "js":"function",
                                 "lsp":"function", "lua":"function", "py":"function"}
    return extMap[ext]
}

func formatDate(v string) string {
    // TODO: Check created data format
    // If given an exact date and time, must use a qualifier: <, >, <=, >=
    // e.g. created:>=2008-04-10T23:59:59Z
    // = is not a valid qualifier. If want equality, omit qualifier
    // e.g created:2008-04-10
    // Also check valid ranges for days, hours, minutes, seconds
    // User can't enter the qualifer as an argument...not sure why, but will need to create anothger flag for them to specify qualifier
    if strings.Compare(string(v[0]), ">") != 0 || strings.Compare(string(v[0:2]), ">=") != 0 || strings.Compare(string(v[0]), "<") != 0 || strings.Compare(string(v[0:2]), "<=") != 0 {
        return "<="+strings.TrimSpace(strings.ToUpper(v))
    } 
    return v
}

func getAuth() (string, string) {
    files, err := ioutil.ReadDir(".auth")

    if err != nil {
        fmt.Println("could not find saved credentials...")
    } else {
        // Assuming only one saved user
        for _, f := range files{
            if strings.Compare(f.Name(), "login") == 0 {
                raw, err2 := ioutil.ReadFile(".auth/"+f.Name())

                if err2 != nil {
                    fmt.Println("error: could not read ./auth/login")
                } else {
                    credRaw := string(raw)
                    cred := strings.Split(credRaw, "\n")
                    return cred[0], cred[1]
                }
            }
        }
    }
    return "", ""
}

func getLatestCheckpoint() GithubSearchResp {
    // Get all checkpoints in dir
    files, err := ioutil.ReadDir("checkpoints")

    if err != nil {
        fmt.Println("error: failed to load checkpoint...")
        log.Fatal(err)
    }

    // Convert to bytes.Buffer
    return loadCheckpoint(mostRecentChkpt(files))
}

func mostRecentChkpt(files []os.FileInfo) string {
    recent := strings.Split(files[0].Name(), "-")
    files  = files[1:]

    // Determine the most recent one
    for _, file := range files {
        date := strings.Split(file.Name(), "-")

        for i := 1; i < len(date); i++ {
            r, _ := strconv.Atoi(recent[i])
            d, _ := strconv.Atoi(date[i])

            if r - d < 0 {
                recent = date
                break
            }
        }
    }

    return strings.Join(recent, "-")
}

func loadCheckpoint(file string) GithubSearchResp {
    raw, err := ioutil.ReadFile("checkpoints/"+file)

    if err != nil {
        fmt.Println(err.Error())
        os.Exit(1)
    }
    var jsonChkpt GithubSearchResp
    json.Unmarshal(raw, &jsonChkpt)
    return jsonChkpt
}

func sleepAndSave(searchResp GithubSearchResp) {
    marshalledStruct, _ := json.Marshal(searchResp)
    saveFile("checkpoints", strings.Join([]string{"ckpt-", time.Now().Format("2006-01-02-15-04-05"), ".json"}, ""), string(marshalledStruct))
    waitTime := getWaitTime()
    fmt.Println("exhausted request quota...hibernating for", waitTime.String())
    time.Sleep(waitTime)
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
/*
    http.Get() doesn't count towards the rate limit unless the URL is an API call
    e.g. Anything beginning with https://api.github.com/
    This is only used for extracting the source code from the repos and checking the function types
*/
func httpGet(url string) string {
    resp, err := http.Get(url)
    
    check(err)
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)

    return string(body)
}

func saveFile(saveDir string, fileName string, fileBody string) {
    f, err := os.Create(strings.Join([]string{saveDir, fileName}, "/"))
    
    check(err)
    defer f.Close()
    _, err = f.WriteString(fileBody)

    check(err)
    f.Sync()
}

func pathExists(dir string)  bool {
    _, err := os.Stat(dir)
    
    if err == nil {
        return true
    } else if os.IsNotExist(err) {
        return false
    } else {
        return true
    }
}
    
func check(e error) {
    if e != nil {
        fmt.Println("fatal: exiting...")
        log.Fatal(e)
    }
}

func findAndClone(cloneDir string, owner string, repoName string, repoURL string, fileName string, inTypeList []string, outTypeList []string, funcNames *[]string, inType *string, outType *string) bool {
    if containsFuncType(fileName, inTypeList, outTypeList, funcNames, inType, outType) {
        cloneRepo(cloneDir, owner, repoName, repoURL)
        return true
    } else {
        source := strings.Join([]string{"tmp", fileName}, "/")
        if pathExists(source) {
            os.Remove(source)
        }
    }
    return false
}

func cloneRepo(cloneDir string, owner string, repoName string, repoURL string) {
    cloneDir = strings.Join([]string{cloneDir, owner, repoName}, "/")
    
    if !pathExists(cloneDir) {
        exec.Command("git", "clone", repoURL, cloneDir).Run()
    }
}

func saveMgoDoc(dbName string, collectionName string, doc Document) bool {
    session, err := mgo.Dial("localhost:27017")
    
    if err != nil {
        panic(err)
    }
    
    defer session.Close()

    collection := session.DB(dbName).C(collectionName)
    err        = collection.Insert(doc)

    if err != nil {
        log.Printf("failed to insert doc into database...\n", doc)
        return false
    }

    return true
}

/*
    Searches source code for functions with inType input and outType output
    Assumes that the file was saved into ./tmp
*/
func containsFuncType(fileName string, inTypeList []string, outTypeList []string, funcNames *[]string, inType *string, outType *string) bool {

    source := strings.Join([]string{"tmp", fileName}, "/")
    fTerm  := strings.Split(fileName, ".")

    ctags := exec.Command("ctags", "-x", "--c-types=f", source)
    grep  := exec.Command("grep", getFuncTerm(fTerm[1]))
    awk   := exec.Command("awk", "{$1=$2=$3=$4=\"\"; print $0}")
    grep.Stdin, _ = ctags.StdoutPipe()
    awk.Stdin, _  = grep.StdoutPipe()
    awkOut, _    := awk.StdoutPipe()
    buff := bufio.NewScanner(awkOut)
    var funcHeaders []string

    _ = grep.Start()
    _ = awk.Start()
    _ = ctags.Run()
    _ = grep.Wait()
    defer awk.Wait()

    for buff.Scan() {    
        funcHeaders = append(funcHeaders, buff.Text()+"\n")
    }

    // fmt.Println(funcHeaders)

    atLeastOne := false

    for _, header := range funcHeaders {
        header = strings.TrimSpace(strings.Split(header, "//")[0])
        parsedHeader := parse.ParseFuncHeader(header, inTypeList, outTypeList, funcNames, inType, outType)

        if parsedHeader {
            atLeastOne = true
            funcHeaders = append(funcHeaders, header)
        }
    }

    return atLeastOne
}

func cleanTmp() {
    os.RemoveAll("tmp")
}

func massivelyClone(queryRespObj GithubSearchResp, dir string) {

    tasks := make(chan *exec.Cmd, runtime.NumCPU())

    var wg sync.WaitGroup

    for i := 0; i < runtime.NumCPU(); i++ {
        wg.Add(1)
        go func() {
            for cmd := range tasks {
                cmd.Run()
            }
            wg.Done()
        }()
    }

    for _, repo := range queryRespObj.Items {
        tasks <- exec.Command("git", "clone", repo.CloneURL, dir)
    }

    close(tasks)
    wg.Wait()
}

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())

    // Connect to MongoDB
    session, err := mgo.Dial("localhost:27017")
    if err != nil {
            panic(err)
    }
    defer session.Close()

    // Choose directory to save repos to
    // Need to add feature to hide credentials as they are entered into the terminal
    reader := bufio.NewReader(os.Stdin)
    fmt.Print("authenticate [y/n]: ")
    auth, _ := reader.ReadString('\n')
    auth = strings.ToLower(strings.Replace(auth, "\n", "", -1))
    didAuth := strings.Compare(auth, "yes") == 0 || strings.Compare(auth, "y") == 0

    un := ""
    pw := ""

    if didAuth {

        un, pw = getAuth()

        if len(un) == 0 && len(pw) == 0 {
            fmt.Print("username: ")
            un, _ = reader.ReadString('\n')
            un = strings.Replace(un, "\n", "", -1)
            fmt.Print("password:")
            pw, _ = reader.ReadString('\n')
            pw = strings.Replace(pw, "\n", "", -1)
            fmt.Print("save credentials[y/n]: ")
            sc, _ := reader.ReadString('\n')
            sc = strings.ToLower(strings.Replace(sc, "\n", "", -1))

            if strings.Compare(sc, "y") == 0 || strings.Compare(sc, "yes") == 0 {
                if !pathExists(".auth") {
                    os.Mkdir(".auth", os.FileMode(0777))
                }
                saveFile(".auth", "login", strings.Join([]string{un, pw}, "\n"))
            }
        }
    }

    // Directory all desired repos will be cloned to    
    fmt.Print("directory to clone all repos to: ")
    dir, _ := reader.ReadString('\n')
    dir = strings.Replace(dir, "\n", "", -1)

    // Make directory if it does not already exist
    if !pathExists(dir) {
        os.Mkdir(dir, os.FileMode(0777))
    }

    // Make a tmp directory for cloning files into when checking their function types
    if !pathExists("tmp") {
        os.Mkdir("tmp", os.FileMode(0777))
    }

    loadCheckpoint := false

    // Make a tmp directory for cloning files into when checking their function types
    if !pathExists("checkpoints") {
        os.Mkdir("checkpoints", os.FileMode(0777))
    } else {
        fmt.Print("use most recent checkpoint [y/n]: ")
        chooseChkpt, _ := reader.ReadString('\n')
        chooseChkpt = strings.ToLower(strings.Replace(chooseChkpt, "\n", "", -1))
        loadCheckpoint = strings.Compare(chooseChkpt, "yes") == 0 || strings.Compare(chooseChkpt, "y") == 0
    }

    // Parameters found here: https://developer.github.com/v3/search/
    flag.String("u", "", "Authenticate with username.")
    flag.String("q", "", "The search terms and any qualifiers.")
    flag.String("in", "", "Qualifies which fields are searched. With this qualifier you can restrict the search to just the repository name, description, readme, or any combination of these.")
    flag.Int("size", 1, "Finds repositories that match a certain size (in kilobytes).")
    flag.Int("forks", 0, "Filters repositories based on the number of forks.")
    flag.Bool("fork", false, "Filters whether forked repositories should be included (true) or only forked repositories should be returned (only).")
    flag.String("created", ">=2007-10-10T00:00:00Z", "Filters repositories based on date of creation.")
    flag.String("pushed", "", "Filters repositories based on date they were last pushed.")
    flag.String("user", "", "Limits searches to a specific user.")
    flag.String("repo", "", "Limits searches to a specific repository.")
    flag.String("language", "Java", "Searches repositories based on the programming language they're written in.")
    flag.Int("stars", 0, "Searches repositories based on the number of stars.")
    flag.String("sort", "", "The sort field. One of stars, forks, or updated. Default: results are sorted by best match.")
    flag.Bool("order", false, "The sort order if sort parameter is provided. One of asc (true) or desc (false). Default: false")
    flag.Bool("all", false, "Search all repos since April 10th, 2008 (GitHub launch date).")
    flag.Bool("mc", false, "Massively clone all repositories found from search.")
    flag.Parse()

    //>=2007-10-10T00:00:00Z
    criteria   := map[string]string{"q":"", "in":"", "size":"", "forks":"", "fork":"", "created":"",
                                   "pushed":"", "updated":"", "user":"", "repo":"", "language":"", "stars":""}
    parameters := map[string]string{"sort":"", "order":"", "per_page":"100"}
    additional := map[string]string{"all":"", "mc":""}

    // Search query queue
    var searchQueue []bytes.Buffer
    var searchResp GithubSearchResp
    var searchQueryLeft  string
    var searchQueryRight string
    var date = Date{}
    var prevDate = time.Date(2008, time.April, 10, 0, 0, 0, 0, time.UTC)

    if !loadCheckpoint {
        var searchQuery bytes.Buffer
        searchQuery.WriteString("https://api.github.com/search/repositories?q=")
        currentFlag := ""

        // Grab all arguments and create the Github search query
        for i, arg := range os.Args {
            arg = strings.ToLower(arg)
            
            // Skip the first item because it will give something 
            // like: /tmp/go-build061196966/command-line-arguments/_obj/exe/gitscrape
            if i > 0 {
                if strings.HasPrefix(arg, "-") {
                    arg = strings.TrimSpace(arg[1:])
                    _, cExists := criteria[arg]
                    _, pExists := parameters[arg] 
                    _, aExists := additional[arg]

                    if cExists || pExists || aExists {
                        currentFlag = arg

                        if strings.Compare(currentFlag, "all") == 0 {
                            // At this exact time and date, there were exactly 1,000 repos written in Java on GitHub
                            var d = GetTime(2009, 3, 12, 21, 0 ,0) 
                            date = Date{"<=", d}
                            criteria["created"] = date.String()
                            additional["all"] = "true"
                        }
                    } else {
                        log.Println(arg," is not a valid flag")
                    }
                } else if len(currentFlag) > 0 {
                    cVal, cExists := criteria[currentFlag]
                       _, pExists := parameters[currentFlag] 
                    arg = strings.TrimSpace(arg)

                    if cExists {
                        criteria[currentFlag] = cVal+" "+arg
                    } else if pExists {
                        parameters[currentFlag] = arg
                    } 
                }
            }
        }

        lang := criteria["language"]
        g := getLangExt(lang)
        if len(g) == 0 {
            log.Fatal("\n", lang, " is not supported\nexiting...")
        }

        criteriaVal := []string{}
        createdStr  := ""

        // Writer search terms first
        searchQuery.WriteString(strings.TrimSpace(criteria["q"]))

        // Write the rest of the search criteria
        for k, v := range criteria {
            if strings.Compare(k, "q") != 0 && len(v) > 0 {
                // Should only have either created flag or pushed set, but not both
                if strings.Compare(k, "created") == 0 {
                    createdStr = "+"+k+":"+strings.TrimSpace(formatDate(v))
                    criteria["pushed"] = "" 
                } else if strings.Compare(k, "pushed") == 0 {
                    createdStr = "+"+k+":"+strings.TrimSpace(formatDate(v))
                    criteria["created"] = ""
                } else {
                    criteriaVal = append(criteriaVal, "+"+k+":"+strings.TrimSpace(v))
                }
            }
        }

        searchQuery.WriteString(strings.Join(criteriaVal, ""))
        searchQueryLeft = searchQuery.String()
        searchQuery.WriteString(createdStr)

        for k, v := range parameters {
            if len(v) > 0 {
                searchQueryRight += "&"+k+"="+v
                searchQuery.WriteString("&"+k+"="+v)
            }
        }

        searchQueue = append(searchQueue, searchQuery)

        fmt.Println(searchQuery.String())

        // searchQueue used in case need to wait for timeout to end
        for 0 < len(searchQueue) {
            searchItem := searchQueue[0]
            searchQueue = searchQueue[:len(searchQueue)-1]
            successfulSearchQuery := search(searchItem.String(), &searchResp, un, pw)

            if !successfulSearchQuery {
                searchQueue = append(searchQueue, searchItem)
                sleepAndSave(searchResp)
            }
        }
    } else {
        searchResp = getLatestCheckpoint()
    }

    // log.Printf("%+v\n", searchResp)
    // Spawn W worker goroutines, W = runtime.NumCPU()
    tasks := make(chan func(), runtime.NumCPU())
    var wg sync.WaitGroup

    for i := 0; i < runtime.NumCPU(); i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for f := range tasks {
                f()
            }
        }()
    }

    inTypeList  := []string{"double", "float", "int", "short", "long", "boolean"}
    outTypeList := []string{"double", "float", "int", "short", "long", "boolean"}

    currentPage := 1
    searchTotal := searchResp.TotalCount - len(searchResp.Items)

    for 0 < len(searchResp.Items) {
        // Dequeues search items, so even if resume search from checkpoint, it won't start from scratch.
        savedRepo       := false
        repo            := searchResp.Items[0]
        searchResp.Items = searchResp.Items[:len(searchResp.Items)-1]
        maybeNext       := StrTimeDate(repo.CreatedAt)

        if maybeNext.After(prevDate) {
            prevDate = maybeNext
        }

        fmt.Println("all: ",additional["all"]," items: ",strconv.Itoa(len(searchResp.Items))," total: ",searchTotal)

        if len(additional["all"]) > 0 && len(searchResp.Items) == 0 {
            var nextQuery bytes.Buffer
            didResetDate := false

            if searchTotal <= 0 {
                // Search another 6 hour interval to continue the search
                date.UTC = prevDate
                nextQuery.WriteString(searchQueryLeft+date.String()+searchQueryRight)
            } else {
                // Search the next page
                currentPage += 1
                nextQuery.WriteString(searchQueryLeft+date.String()+searchQueryRight+"&page="+strconv.Itoa(currentPage))
                didResetDate = true
            }

            searchQueue = append(searchQueue, nextQuery)

            for 0 < len(searchQueue) {
                searchItem            := searchQueue[0]
                searchQueue            = searchQueue[:len(searchQueue)-1]
                successfulSearchQuery := search(searchItem.String(), &searchResp, un, pw)

                if !successfulSearchQuery {
                    searchQueue = append(searchQueue, searchItem)
                    sleepAndSave(searchResp)
                }
            }

            // Calculate remaining number of search items to determine if should go to next page
            // If starts searching from a new date-time, then total_count will change and be updated accordingly
        	if didResetDate {
        		searchTotal = searchResp.TotalCount
        	}

        	pageTotal := len(searchResp.Items)
            searchTotal -= pageTotal
        }

        // url for this particular repo
        var contentQuery bytes.Buffer
        contentQuery.WriteString("https://api.github.com/repos/")
        contentQuery.WriteString(strings.Join([]string{repo.Owner.Login, repo.Name, "contents"}, "/"))
        
        // Get the contents in the home directory of this repo
        // fmt.Println("next repo")
        var contentResp []GithubContentResp
        contentQuerySuccess := search(contentQuery.String(), &contentResp, un, pw)

        if !contentQuerySuccess {
            searchResp.Items = append(searchResp.Items, repo)
            sleepAndSave(searchResp)
        }

        // BFS on this repo
        for 0 < len(contentResp) {

            // Dequeue
            cont       := contentResp[0]
            contentResp = contentResp[1:]

            tasks <- func() {
                if strings.Compare(cont.ContentType, "file") == 0 && strings.HasSuffix(cont.Name, getLangExt(criteria["language"])) {
                    saveFile("tmp", cont.Name, httpGet(cont.DownloadURL))

                    // Should prompt user for desired function type and even give them a feature for specifying type of search
                    // Maybe they don't want to search for function types
                    var funcNames []string
                    var inType string
                    var outType string

                    didFind   := findAndClone(dir, repo.Owner.Login, repo.Name, repo.CloneURL, cont.Name, inTypeList, outTypeList, 
                                              &funcNames, &inType, &outType)
                    if didFind {

                        if !savedRepo {
                            savedRepo  = saveMgoDoc("github_repos", "repository",
                                                     RepoDoc{Owner:repo.Owner.Login, RepoName:repo.Name, RepoURL:repo.CloneURL, 
                                                     RepoSizeKB:repo.SizeKB, RepoLang:repo.Language, 
                                                     FilePath:strings.Join([]string{dir, repo.Owner.Login, repo.Name}, "/")})
                        }

                        if savedRepo {
                            session, err := mgo.Dial("localhost:27017")

                            if err != nil {
                                panic(err)
                            }

                            defer session.Close()
                            c := session.DB("github_repos").C("repository")

                            repoMgoDoc := RepoDoc{}
                            err = c.Find(bson.M{"reponame":repo.Name}).One(&repoMgoDoc)

                            if err != nil {
                                fmt.Println("error: could not find repository...")
                            }

                            // fmt.Println(funcNames)

                            for _, fname := range funcNames {
                                savedFunc := saveMgoDoc("github_repos", "function", 
                                                     FuncDoc{RepoId:repoMgoDoc.getObjectIdStr(), RawURL:cont.DownloadURL,
                                                     FileName:cont.Name, FuncName:fname, InputType:inType, OutputType:outType,
                                                     ASTID:"", CFGID:""})
                                if !savedFunc {
                                    log.Fatal("Could not save function to mgo")
                                }
                            }
                        }
                    }
                } else if strings.Compare(cont.ContentType, "dir") == 0 {
                    // Construct the url to search this sub-directory
                    // May need to prepend "'User-Agent: <username>'" to api call for correctness
                    //   Refer to GitHub api doc for more info.
                    var contentDir bytes.Buffer
                    contentDir.WriteString("https://api.github.com/repos/")
                    contentDir.WriteString(strings.Join([]string{repo.Owner.Login, repo.Name, "contents", cont.Name}, "/"))

                    var subdirContentResp []GithubContentResp
                    contentDirSuccess := search(contentDir.String(), &subdirContentResp, un, pw)

                    if !contentDirSuccess {
                        contentResp = append(contentResp, cont)
                        sleepAndSave(searchResp)
                    }

                    // Enqueue contents of sub-directory
                    for  _, subdirCont := range subdirContentResp {
                        if strings.Compare(subdirCont.ContentType, "dir") == 0 {
                            // If a directory, need to prepend current dir's name to its subdir's name so it can search correctly
                            // in the outter loop above
                            subdirCont.Name = strings.Join([]string{cont.Name, subdirCont.Name}, "/")
                        }
                        contentResp = append(contentResp, subdirCont)
                    }
                }
            }
        }
    }

    close(tasks)
    wg.Wait()

    cleanTmp()
}