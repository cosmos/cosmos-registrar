package prompts

import (
	"fmt"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

// Confirm - prompt the user for a yes/no answer
func Confirm(defaultAnswer bool, q string, v ...interface{}) (answer bool) {
	prompt := &survey.Confirm{
		Message: fmt.Sprintf(q, v...),
		Default: defaultAnswer,
	}
	err := survey.AskOne(prompt, &answer)
	if err != nil {
		answer = defaultAnswer
	}
	return
}

// InputRequired - ask user for mandatory input
func InputRequired(q string, v ...interface{}) (answer string, err error) {
	prompt := &survey.Input{
		Message: fmt.Sprintf(q, v...),
	}
	err = survey.AskOne(prompt, &answer, survey.WithValidator(survey.Required))
	return
}

// InputOrDefault - ask user for mandatory input
func InputOrDefault(defaultValue, q string, v ...interface{}) (answer string, err error) {
	prompt := &survey.Input{
		Message: fmt.Sprintf(q, v...),
		Default: defaultValue,
	}
	err = survey.AskOne(prompt, &answer)
	return
}

// Password - ask user for password input
func Password(q string, v ...interface{}) (password string, err error) {
	prompt := &survey.Password{
		Message: fmt.Sprintf(q, v...),
	}
	err = survey.AskOne(prompt, &password, survey.WithValidator(survey.Required))
	return
}

// PrettyMap - pretty print a map
func PrettyMap(data map[string]interface{}) {
	var settings sort.StringSlice
	for k, v := range data {
		if strings.Contains(k, "token") {
			v = "********"
		}
		settings = append(settings, fmt.Sprintf("%-22s: %s", k, v))
	}
	sort.Sort(settings)
	println(strings.Join(settings, "\n"))
}

// Option - to be used for a select input
type Option struct {
	Label string
	Func  func() error
}

// NewOption - create a new option
func NewOption(label string, op func() error) Option {
	return Option{
		Label: label,
		Func:  op,
	}
}

// Select - render a select field
func Select(q string, options ...Option) (err error) {
	items := make([]string, len(options))
	for i, o := range options {
		items[i] = o.Label
	}
	var index int
	prompt := &survey.Select{
		Message: q,
		Options: items,
	}
	err = survey.AskOne(prompt, &index)
	if err != nil {
		return
	}
	err = options[index].Func()
	return

}
