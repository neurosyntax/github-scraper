package parse

import (
	"strings"
)

func ParseFuncHeader (header string, inTypeList []string, outTypeList []string, funcNames *[]string, inType *string, outType *string) bool {
	split := strings.Split(header, "(")
	atLeastOne := false
	        
	if len(split) == 2 {
	    // Check return type
	    hasRtn        := false
	    nonParameters := strings.Split(split[0], " ")

	    if len(nonParameters) > 2 {

	        // Assuming Java syntax. 
	        // Return type is always second keyword after the visibility modifer in the function header.
	        rtnType := nonParameters[1]

	        for _, r := range outTypeList {
	            if strings.Compare(rtnType, r) == 0 {
	                hasRtn   = true
	                *outType = rtnType 
	                break
	            }
	        }
	    }

	    if hasRtn {
	        hasParameters := false
	        parameters    := strings.Split(strings.Split(split[1], ")")[0], " ")
	        var trackParams []string
	        // fmt.Println(parameters)

	        // Check function parameters
	        for i, t := range parameters {
	            if i % 2 == 0 {
	                for _, s := range inTypeList {
	                    if strings.Compare(t, s) == 0 {
	                        hasParameters = true
	                        trackParams = append(trackParams, s)
	                    } else {
	                    	*outType = ""
	                    	*inType  = ""
	                    	return false
	                    }

	                }
	            }

	            if !hasParameters {
	                break
	            } else {
	                *inType = strings.Join(trackParams, " ")
	            }
	        }

	        if hasParameters && hasRtn {
	        	atLeastOne = true
	            *funcNames = append(*funcNames, header)
	        }
	    }
	}

	return atLeastOne
}