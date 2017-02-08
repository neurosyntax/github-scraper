/*
    query.go

    A tool for scraping repositories from GitHub and extracting source code function information using Golang and GitHub API v3.
    
    Author: Justin Chen
    12.19.2016

    Boston University 
    Computer Science

    Dependencies:        exuberant ctags, and mongodb driver for go (http://labix.org/mgo)
    Operating systems:   GNU Linux, OS X
    Supported languages: C, C++, C#, Erlang, Lisp, Lua, Java, Javascript, and Python
*/

package query

import (
    "utils"
)

//>=2007-10-10T00:00:00Z
var criteria   = map[string]string{"q":"", "in":"", "size":"", "forks":"", "fork":"", "created":"",
                                   "pushed":"", "updated":"", "user":"", "repo":"", "language":"", "stars":""}
var parameters = map[string]string{"sort":"", "order":"", "per_page":"100", "all":"", "mc":""}

type Query struct {
	Left  string
	Right string
	Date  string
	Cred  Credentials
    Criteria map[string]string
    Parameters map[string]string
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
