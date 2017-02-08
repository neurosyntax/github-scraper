/*
    parse.go

    A tool for scraping repositories from GitHub and extracting source code function information using Golang and GitHub API v3.
    
    Author: Justin Chen
    12.19.2016

    Boston University 
    Computer Science

    Dependencies:        exuberant ctags, and mongodb driver for go (http://labix.org/mgo)
    Operating systems:   GNU Linux, OS X
    Supported languages: C, C++, C#, Erlang, Lisp, Lua, Java, Javascript, and Python
*/

package parse

import (
	"strings"
)
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

func ParseFuncHeader (header string, inTypeList []string, outTypeList []string, funcNames *[]string, inType *string, outType *string) bool {
	split := strings.Split(header, "(")
	atLeastOne := false
	        
	if len(split) == 2 {
	    // Check return type
	    hasRtn        := false
	    nonParameters := strings.Split(split[0], " ")

	    if len(nonParameters) > 2 {

	        for _, t := range nonParameters {
	        	// fmt.Println(t)
		        for _, r := range outTypeList {
		            if strings.Compare(t, r) == 0 {
		                hasRtn   = true
		                *outType = t 
		                break
		            }
		        }
		        if hasRtn {
		        	break
		        }
		    }
	    }

	    // fmt.Println(hasRtn)

	    if hasRtn {
	        hasParameters := false
	        parameters    := strings.Split(strings.Split(split[1], ")")[0], " ")
	        // fmt.Println(parameters)
	        var trackParams []string
	        // fmt.Println(parameters)

	        // Check function parameters
	        for i, t := range parameters {
	            if i % 2 == 0 {
	                for _, s := range inTypeList {
	                    if strings.Compare(t, s) == 0 {
	                        hasParameters = true
	                        trackParams = append(trackParams, s)
	                    }
	                }
	            }

	            // fmt.Println("hasParameters: ",hasParameters)

	            if !hasParameters {
	                break
	            } else {
	                *inType = strings.Join(trackParams, " ")
	            }
	        }

	        if hasParameters {
	        	atLeastOne = true
	            *funcNames = append(*funcNames, header)
	        } else {
            	*outType = ""
            	*inType  = ""
	        }
	    }
	}

	return atLeastOne
}

/*
    Searches source code for functions with inType input and outType output
    Assumes that the file was saved into ./tmp
*/
func containsFuncType(fileName string, inTypeList []string, outTypeList []string, funcNames *[]string, inType *string, outType *string) bool {

    source := strings.Join([]string{"tmp", fileName}, "/")
    fName  := strings.Split(fileName, ".")
    atLeastOne := false

    if len(fName) <= 1 {
        return atLeastOne
    }

    ctags := exec.Command("ctags", "-x", "--c-types=f", source)
    grep  := exec.Command("grep", getFuncTerm(fName[1]))
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