/*
    retrieve.go

    A tool for scraping repositories from GitHub and extracting source code function information using Golang and GitHub API v3.
    
    Author: Justin Chen
    12.19.2016

    Boston University 
    Computer Science

    Dependencies:        exuberant ctags, and mongodb driver for go (http://labix.org/mgo)
    Operating systems:   GNU Linux, OS X
    Supported languages: C, C++, C#, Erlang, Lisp, Lua, Java, Javascript, and Python
*/

package retrieve

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