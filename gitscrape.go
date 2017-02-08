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
    "parse"
    "search"
    "query"
    "utils"
    "data"
    "retrieve"
)

func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())
    session        := data.connectDB()
    cred           := utils.initAuth()
    dir            := utils.initSaveDir()
    loadCheckpoint := utils.useCheckpoint()

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
    currentSearch := ""
    currentPage   := 1
    // currentRepo   := ""

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
            successfulSearchQuery := search(searchItem.String(), &searchResp, cred.User, cred.Pass)

            if !successfulSearchQuery {
                searchQueue = append(searchQueue, searchItem)
                sleepAndSave(searchItem.String(), 1, "", searchResp, date, prevDate)
            }
        }
    } else {
        checkpt      := getLatestCheckpoint()
        searchResp    = checkpt.SearchResp
        currentSearch = checkpt.LastSearch
        currentPage   = checkpt.CurrentPage
        // currentRepo   = checkpt.CurrentRepo
        date          = checkpt.CurrentDate
        prevDate      = checkpt.NextDate
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

    searchTotal   := searchResp.TotalCount - len(searchResp.Items)

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
                successfulSearchQuery := search(searchItem.String(), &searchResp, cred.User, cred.Pass)

                if !successfulSearchQuery {
                    searchQueue = append(searchQueue, searchItem)
                    sleepAndSave(searchItem.String(), currentPage, "", searchResp, date, prevDate)
                }
            }

            currentSearch = nextQuery.String()

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
        contentQuerySuccess := search(contentQuery.String(), &contentResp, cred.User, cred.Pass)

        if !contentQuerySuccess {
            searchResp.Items = append(searchResp.Items, repo)
            sleepAndSave(currentSearch, currentPage, contentQuery.String(), searchResp, date, prevDate)
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
                        repoId := hash(repo.Owner.Login+repo.Name)

                        // Will try to insert repo to repository collection and return false if could not
                        // Will not create duplicates b/c hash is deterministic and owner+reponame is unique
                        saveMgoDoc("github_repos", "repository",
                                    RepoDoc{repoId, Owner:repo.Owner.Login, RepoName:repo.Name, RepoURL:repo.CloneURL, 
                                    RepoSizeKB:repo.SizeKB, RepoLang:repo.Language, 
                                    FilePath:strings.Join([]string{dir, repo.Owner.Login, repo.Name}, "/")})

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

                        for _, fname := range funcNames {
                            savedFunc := saveMgoDoc("github_repos", "function", 
                                                 FuncDoc{RepoId:repoId, RawURL:cont.DownloadURL,
                                                 FileName:cont.Name, FuncName:fname, InputType:inType, OutputType:outType,
                                                 ASTID:"", CFGID:""})
                            if !savedFunc {
                                log.Fatal("Could not save function to mgo")
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
                    contentDirSuccess := search(contentDir.String(), &subdirContentResp, cred.User, cred.Pass)

                    if !contentDirSuccess {
                        contentResp = append(contentResp, cont)
                        sleepAndSave(currentSearch, currentPage, contentDir.String(), searchResp, date, prevDate)
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