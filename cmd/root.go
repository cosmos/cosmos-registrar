/*
Package cmd ...
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/libs/log"

	registrar "github.com/jackzampolin/cosmos-registrar/pkg/config"
	"github.com/jackzampolin/cosmos-registrar/pkg/prompts"
)

var (
	cfgFile       string
	debug         bool
	config        *registrar.Config
	logger        = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	noInteraction = false
)

func init() {
	cobra.OnInitialize(initConfig)

	cobra.EnableCommandSorting = false
	rootCmd.SilenceUsage = true
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file")

	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")
	rootCmd.Flags().BoolVarP(&noInteraction, "no-interactive", "y", false, "Run commands non interactively")

	rootCmd.AddCommand(
		configCmd(),
		updateCmd,
		getVersionCmd(),
	)
}

var rootCmd = &cobra.Command{
	Use:   "registrar",
	Short: "registers data aiding in chain service discovery (peers seeds etc...) in github repo",
	Run:   interact,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func interact(cmd *cobra.Command, args []string) {

	for {
		prompts.Select("what shall we do today?",
			prompts.NewOption("Register a new ChainID", func() (err error) {
				// ask use to create a fork first
				println(`
Good choice, following this process you will submit a 
pull request to the chain IDs registry hosted on GitHub:
` + config.RegistryRoot + `

The first step is to create a fork of the registry using this link:
` + fmt.Sprintf("%s/fork", config.RegistryRoot))

				if ok := prompts.Confirm("Go ahead and confirm when you have done so", "Y"); !ok {
					println("please create the fork before continuing")
					return
				}
				println(`
Next enter the rpc url for a node of your network  
eg http://10.0.0.1:26657
`)
				// TODO add rpc address prompt
				rpc, err := prompts.InputRequired("rpc address")

				claim(cmd, []string{rpc})
				return
			}),
			prompts.NewOption("Update a Chain ID you control", func() (err error) {
				println("coming soon")
				return
			}),
			prompts.NewOption("Exit", func() (err error) {
				println("goodbye!")
				os.Exit(0)
				return
			}),
		)
	}

}
