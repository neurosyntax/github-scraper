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

/*type RepoMgoDoc struct {
    RepoDoc
    Id bson.ObjectId `json:"id" bson:"_id,omitempty"`
}*/

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

func main() {

    // Load programming language terminology for functions
    // loadFuncTerms()

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
        fmt.Print("username: ")
        un, _ = reader.ReadString('\n')
        un = strings.Replace(un, "\n", "", -1)
        fmt.Print("password:")
        pw, _ = reader.ReadString('\n')
        pw = strings.Replace(pw, "\n", "", -1)
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
    flag.String("created", "", "Filters repositories based on date of creation.")
    flag.String("updated", "", "Filters repositories based on date they were last updated.")
    flag.String("user", "", "Limits searches to a specific user.")
    flag.String("repo", "", "Limits searches to a specific repository.")
    flag.String("language", "", "Searches repositories based on the programming language they're written in.")
    flag.Int("stars", 0, "Searches repositories based on the number of stars.")
    flag.String("sort", "", "The sort field. One of stars, forks, or updated. Default: results are sorted by best match.")
    flag.Bool("order", false, "The sort order if sort parameter is provided. One of asc (true) or desc (false). Default: false")
    flag.Bool("mc", false, "Massively clone all repositories found from search.")
    flag.Parse()

    // Search query queue
    var searchQueue []bytes.Buffer
    var searchResp GithubSearchResp

    if !loadCheckpoint {
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

        searchQueue = append(searchQueue, searchQuery)

        // searchQueue used in case need to wait for timeout to end
        for 0 < len(searchQueue) {
            searchItem := searchQueue[0]
            searchQueue = searchQueue[:len(searchQueue)-1]
            successfulSearchQuery := search(searchItem, &searchResp, un, pw)
            // log.Printf("%+v\n", searchResp)
            if !successfulSearchQuery {
                searchQueue = append(searchQueue, searchItem)
                sleepAndSave(searchResp)
            }
        }
    } else {
        searchResp = getLatestCheckpoint()
    }

    // Spawn W worker goroutines, W = runtime.NumCPU()
    tasks := make(chan func(), runtime.NumCPU())
    var wg sync.WaitGroup

    for i := 0; i < runtime.NumCPU(); i++ {
        wg.Add(1)
        go func() {
            for f := range tasks {
                f()
            }
            wg.Done()
        }()
    }

    inTypeList  := []string{"double", "float", "int", "short", "long", "boolean"}
    outTypeList := []string{"double", "float", "int", "short", "long", "boolean"}

    for 0 < len(searchResp.Items) {

        // Dequeues search items, so even if resume search from checkpoint, it won't start from scratch.
        repo := searchResp.Items[0]
        searchResp.Items = searchResp.Items[1:]

        // url for this particular repo
        var contentQuery bytes.Buffer
        contentQuery.WriteString("https://api.github.com/repos/")
        contentQuery.WriteString(strings.Join([]string{repo.Owner.Login, repo.Name, "contents"}, "/"))
        
        // Get the contents in the home directory of this repo
        fmt.Println("next repo")
        var contentResp []GithubContentResp
        contentQuerySuccess := search(contentQuery, &contentResp, un, pw)
        // log.Printf("%+v\n", contentResp)
        if !contentQuerySuccess {
            searchResp.Items = append(searchResp.Items, repo)
            sleepAndSave(searchResp)
        }

        // BFS on this repo
        for 0 < len(contentResp) {

            // Dequeue
            cont       := contentResp[0]
            contentResp = contentResp[1:]

            if strings.Compare(cont.ContentType, "file") == 0 && strings.HasSuffix(cont.Name, ".java") {
                saveFile("tmp", cont.Name, httpGet(cont.DownloadURL))

                // Should prompt user for desired function type and even give them a feature for specifying type of search
                // Maybe they don't want to search for function types
                tasks <- func() {
                    var funcNames []string
                    var inType string
                    var outType string

                    didFind   := findAndClone(dir, repo.Owner.Login, repo.Name, repo.CloneURL, cont.Name, inTypeList, outTypeList, &funcNames, &inType, &outType)
                    if didFind {
                        savePath := strings.Join([]string{dir, repo.Owner.Login, repo.Name}, "/")
                        repoDoc  := RepoDoc{Owner:repo.Owner.Login, RepoName:repo.Name, RepoURL:repo.CloneURL, RepoSizeKB:repo.SizeKB, RepoLang:repo.Language, FilePath:savePath}
                        didSave  := saveMgoDoc("github_repos", "repository", &repoDoc)
                        if didSave {
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

                            fmt.Println(funcNames)
                            // fmt.Println(repo)

                            for _, fname := range funcNames {
                                funcDoc := FuncDoc{RepoId:repoMgoDoc.getObjectIdStr(), RawURL:cont.DownloadURL, FileName:cont.Name, FuncName:fname, InputType:inType, OutputType:outType, ASTID:"", CFGID:""}
                                didSave = saveMgoDoc("github_repos", "function", funcDoc)
                                
                                if !didSave {
                                    log.Fatal("Could not save function to mgo")
                                }
                            }
                        }
                    }
                }

            } else if strings.Compare(cont.ContentType, "dir") == 0 {
                // Construct the url to search this sub-directory
                var contentDir bytes.Buffer
                contentDir.WriteString("https://api.github.com/repos/")
                contentDir.WriteString(strings.Join([]string{repo.Owner.Login, repo.Name, "contents", cont.Name}, "/"))

                // fmt.Println(contentDir.String())
                var subdirContentResp []GithubContentResp

                contentDirSuccess := search(contentDir, &subdirContentResp, un, pw)
                if !contentDirSuccess {
                    contentResp = append(contentResp, cont)
                    sleepAndSave(searchResp)
                }
                // log.Printf("%+v\n", subdirContentResp)

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
    }

    close(tasks)
    wg.Wait()

    cleanTmp()
}

func getLatestCheckpoint() GithubSearchResp {
    // Get all checkpoints in dir
    files, err := ioutil.ReadDir("checkpoints")

    if err != nil {
        fmt.Println("failed to load checkpoint...")
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

    return strings.Join(recent, "")
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
    t := time.Now()
    marshalledStruct, _ := json.Marshal(searchResp)
    saveFile("checkpoints", strings.Join([]string{"ckpt-", t.Format("2006-01-02-15-04-05"), ".json"}, ""), string(marshalledStruct))
    fmt.Println("exhausted request quota...hibernating for an hour...")
    time.Sleep(3600000 * time.Millisecond)
}

func search(query bytes.Buffer, queryResp interface{}, username string, password string) bool {
    client := &http.Client{}
    req, err := http.NewRequest("GET", query.String(), nil)
    
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

func loadFuncTerms() {
    // Read in txt file
    // Loop and build string-to-string map
    // Return map
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
    // Determine file type using extension in fileName
    // make a separate function with a lookup table for extension types
    // then return what ctags refers to functions as in that language
    // e.g. functions are referred to as members in Python and methods in Java
    // then replace the second string below in the grep command

    source := strings.Join([]string{"tmp", fileName}, "/")

    ctags := exec.Command("ctags", "-x", "--c-types=f", source)
    grep  := exec.Command("grep", "method")//getFuncRef(fileName))
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
