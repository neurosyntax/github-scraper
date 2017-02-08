/*
    utils.go

    A tool for scraping repositories from GitHub and extracting source code function information using Golang and GitHub API v3.
    
    Author: Justin Chen
    12.19.2016

    Boston University 
    Computer Science

    Dependencies:        exuberant ctags, and mongodb driver for go (http://labix.org/mgo)
    Operating systems:   GNU Linux, OS X
    Supported languages: C, C++, C#, Erlang, Lisp, Lua, Java, Javascript, and Python
*/

package utils

import (
	"time"
	"search"
	"bufio"
	"os"
	"strings"
)

type Checkpoint struct {
    LastSearch string
    CurrentPage int
    CurrentRepo string
    SearchResp GithubSearchResp
    CurrentDate Date
    NextDate time.Time
}

type Credentials struct {
	User string
	Pass string
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

func initSaveDir() string {
	// Directory all desired repos will be cloned to
    reader := bufio.NewReader(os.Stdin)    
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

    return dir
}

func initAuth() *Credentials {
	// Need to add feature to hide credentials as they are entered into the terminal
    reader := bufio.NewReader(os.Stdin)
    fmt.Print("authenticate [y/n]: ")
    auth, _ := reader.ReadString('\n')
    auth = strings.ToLower(strings.Replace(auth, "\n", "", -1))
    didAuth := strings.Compare(auth, "yes") == 0 || strings.Compare(auth, "y") == 0

    un := ""
    pw := ""

    if didAuth {
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
    	} else {
    		un, pw = loadAuth()
    	}
    }

    return &Credentials{un, pw}
}

func loadAuth() (string, string) (string, string) {
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

func useCheckpoint() bool {
	// Make a tmp directory for cloning files into when checking their function types
    if !pathExists("checkpoints") {
        os.Mkdir("checkpoints", os.FileMode(0777))
    }

    reader := bufio.NewReader(os.Stdin)
    fmt.Print("use most recent checkpoint [y/n]: ")
    chooseChkpt, _ := reader.ReadString('\n')
    chooseChkpt = strings.ToLower(strings.Replace(chooseChkpt, "\n", "", -1))
    return strings.Compare(chooseChkpt, "yes") == 0 || strings.Compare(chooseChkpt, "y") == 0
}

func getLatestCheckpoint() Checkpoint {
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

func loadCheckpoint(file string) Checkpoint {
    raw, err := ioutil.ReadFile("checkpoints/"+file)

    if err != nil {
        fmt.Println(err.Error())
        os.Exit(1)
    } else {
        fmt.Println("loaded checkpoint: ",raw)
    }

    var jsonChkpt Checkpoint
    json.Unmarshal(raw, &jsonChkpt)
    return jsonChkpt
}

func sleepAndSave(lastSearch string, currentPage int, currentRepo string, searchResp GithubSearchResp, currDate Date, prevDate time.Time) {
    checkpoint          := Checkpoint{lastSearch, currentPage, currentRepo, searchResp, currDate, prevDate}
    marshalledStruct, _ := json.Marshal(checkpoint)
    saveFile("checkpoints", strings.Join([]string{"ckpt-", time.Now().Format("2006-01-02-15-04-05"), ".json"}, ""), string(marshalledStruct))
    waitTime := getWaitTime()
    fmt.Println("exhausted request quota...hibernating for", waitTime.String())
    time.Sleep(waitTime)
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

func cleanTmp() {
    os.RemoveAll("tmp")
}