package utils

import (
	"fmt"
	"os"
)

// AbortIfError abort command if there is an error
func AbortIfError(err error, message string, v ...interface{}) {
	if err == nil {
		return
	}
	fmt.Printf(message, v...)
	fmt.Println()
	os.Exit(1)
}

// ContainsStr tells whenever a list of strring contains a specific string
func ContainsStr(elements *[]string, needle *string) bool {
	for _, s := range *elements {
		if *needle == s {
			return true
		}
	}
	return false
}
