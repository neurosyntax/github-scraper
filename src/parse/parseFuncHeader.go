package parse

import (
	// "fmt"
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