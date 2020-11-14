package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	// Version defines the application version (defined at compile time)
	Version = ""
	// Commit defines the application commit hash (defined at compile time)
	Commit = ""
	// TMVersion defines the tendermint commit hash (defined at compile time)
	TMVersion = ""
)

type versionInfo struct {
	Version   string `json:"version" yaml:"version"`
	Commit    string `json:"commit" yaml:"commit"`
	TMVersion string `json:"tm-version" yaml:"tm-version"`
	Go        string `json:"go" yaml:"go"`
}

func getVersionCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "Print registrar version info",
		RunE: func(cmd *cobra.Command, args []string) error {
			verInfo := versionInfo{
				Version:   Version,
				Commit:    Commit,
				TMVersion: TMVersion,
				Go:        fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH),
			}
			bz, err := yaml.Marshal(&verInfo)
			if err != nil {
				return err
			}
			fmt.Println(string(bz))
			return nil
		},
	}
	return versionCmd
}
