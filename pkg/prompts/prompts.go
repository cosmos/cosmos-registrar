package prompts

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackzampolin/cosmos-registrar/pkg/config"
	"github.com/manifoldco/promptui"
)

// Setup guides the user to setup their environmant
func Setup() (config config.Config, err error) {
	println(`Welcome to the Cosmos registrar tool. 
	This tool will allow you to publicly claim a name for your 
	cosmos based chain.`)

	println(`To complete the setup you need a Github account and 
	network connectivity to a node of your chain.`)
	// fail if they are not available
	if goOn := Confirm("do you have them available?", "Y"); !goOn {
		println("please make sure you get them and the run the setup again")
		return
	}
	// next get the github user
	config.GitName, err = InputRequired("enter your github name")
	if err != nil {
		println(err.Error())
	}
	// now get the github token
	println(`the next step is to enter a github personal token, 
if you don't have one you can get it from 
https://github.com/settings/tokens
make sure that you select the permission repo > public_repo`)
	config.GithubAccessToken, err = InputRequired("token")
	// now get the github token
	println("what is a node rpc address for the chain you want to register (eg. http://10.0.0.1:26657)")
	config.RPCAddr, err = InputRequired("rpc address: ")
	println("the setup is completed")
	return
}

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
