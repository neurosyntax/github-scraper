/*
    data.go

    A tool for scraping repositories from GitHub and extracting source code function information using Golang and GitHub API v3.
    
    Author: Justin Chen
    12.19.2016

    Boston University 
    Computer Science

    Dependencies:        exuberant ctags, and mongodb driver for go (http://labix.org/mgo)
    Operating systems:   GNU Linux, OS X
    Supported languages: C, C++, C#, Erlang, Lisp, Lua, Java, Javascript, and Python
*/

package data

import (
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
)

/*
	Connect to MongoDB and return the session
*/
func connectDB() *Session {
	// Connect to MongoDB
    session, err := mgo.Dial("localhost:27017")
    if err != nil {
            panic(err)
    }
    defer session.Close()
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