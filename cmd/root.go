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
	"os"

	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/libs/log"

	registrar "github.com/jackzampolin/cosmos-registrar/pkg/config"
)

var (
	cfgFile       string
	config        *registrar.Config
	logger        = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	noInteraction = false
)

func init() {
	cobra.OnInitialize(initConfig)

	cobra.EnableCommandSorting = false
	rootCmd.SilenceUsage = true
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file")

	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
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
	println("welcome")
}
