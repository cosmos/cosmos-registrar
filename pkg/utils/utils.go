package utils

import (
	"encoding/json"
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
func ContainsStr(elements *[]string, needle string) bool {
	for _, s := range *elements {
		if needle == s {
			return true
		}
	}
	return false
}

// FromJSON unmarshal a json file to an inferface
func FromJSON(path string, v interface{}) (err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	if err = json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("unmarshaling data: %s", err)
	}
	return
}

// ToJSON write data to a json file
func ToJSON(path string, data interface{}) (err error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	err = os.WriteFile(path, raw, 0x600)
	return
}

// PathExists tells if a path exists
func PathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}
