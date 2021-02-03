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
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	config     *registrar.Config
	cfgFlag    = "config"
	defaultCfg = os.ExpandEnv("$HOME/.registrar.yaml")
	logger     = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
)

func init() {
	cobra.EnableCommandSorting = false
	rootCmd.SilenceUsage = true
	rootCmd.PersistentFlags().StringVar(&cfgFile, cfgFlag, defaultCfg, "config file (default is $HOME/.registrar.yaml)")
	if err := viper.BindPFlag(cfgFlag, rootCmd.Flags().Lookup(cfgFlag)); err != nil {
		panic(err)
	}
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.AddCommand(
		configCmd(),
		updateCmd,
		getVersionCmd(),
	)
}

var rootCmd = &cobra.Command{
	Use:   "registrar",
	Short: "registers data aiding in chain service discovery (peers seeds etc...) in github repo",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// reads `.registrar.yaml` into `var config *Config` before each command
	rootCmd.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		return initConfig(rootCmd)
	}
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
