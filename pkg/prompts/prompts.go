package prompts

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/manifoldco/promptui"
)

// Confirm - prompt the user for a yes/no answer
func Confirm(f, defaultAnswer string, v ...interface{}) bool {
	prompt := promptui.Prompt{
		Label:     fmt.Sprintf(f, v...),
		IsConfirm: true,
		Default:   defaultAnswer,
	}
	result, err := prompt.Run()
	if err != nil {
		return false
	}
	return strings.ToLower(strings.TrimSpace(result)) == "y"
}

// InputRequired - ask user for mandatory input
func InputRequired(q string, v ...interface{}) (string, error) {
	return input(func(v string) (err error) {
		if strings.TrimSpace(v) == "" {
			err = errors.New("the answer cannot be empty")
		}
		return
	}, q, v...)
}

func input(validator func(v string) error, q string, v ...interface{}) (res string, err error) {
	prompt := promptui.Prompt{
		Label:    fmt.Sprintf(q, v...),
		Validate: validator,
	}
	res, err = prompt.Run()
	return
}

// PrettyMap - pretty print a map
func PrettyMap(data map[string]interface{}) {
	var settings sort.StringSlice
	for k, v := range data {
		settings = append(settings, fmt.Sprintf("%-22s: %s", k, v))
	}
	sort.Sort(settings)
	println(strings.Join(settings, "\n"))
}
