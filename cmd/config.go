package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	registrar "github.com/jackzampolin/cosmos-registrar/pkg/config"
	"github.com/spf13/cobra"

	"gopkg.in/yaml.v2"
)

// initConfig reads in config file and ENV variables if set.
func initConfig(cmd *cobra.Command) (err error) {
	config = &registrar.Config{}
	if _, err = os.Stat(cfgFile); err == nil {
		file, err := ioutil.ReadFile(cfgFile)
		if err != nil {
			return fmt.Errorf("Error reading config: %s", err)
		}
		if err = yaml.Unmarshal(file, config); err != nil {
			return fmt.Errorf("Error unmarshalling config: %s", err)
		}
		return nil
	}
	fmt.Println("no config file", cfgFile)
	return nil
}

func defaultConfig() []byte {
	c := &registrar.Config{
		RPCAddr:            "http://localhost:26657",
		ChainID:            "cosmoshub-3",
		BuildRepo:          "https://github.com/cosmos/gaia",
		BinaryName:         "gaiad",
		BuildCommand:       "make install",
		BuildVersion:       "v2.0.13",
		GithubAccessToken:  "get yours at https://github.com/settings/tokens",
		RegistryRoot:       "https://github.com/cosmos/registry",
		RegistryForkName:   "cosmos-registry",
		RegistryRootBranch: "main",
		GitName:            "Your name goes here",
		GitEmail:           "your@email.here",
		CommitMessage:      "update roots of trust",
		Workspace:          "/tmp",
	}
	config = c
	return c.MustYAML()
}

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config",
		Aliases: []string{"cfg"},
		Short:   "manage configuration file",
	}

	cmd.AddCommand(
		configInitCmd(),
		configDeleteCmd(),
		configShowCmd(),
		configEditCmd(),
	)

	return cmd
}

// Command for inititalizing an empty config at the --home location
func configInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "init",
		Aliases: []string{"i"},
		Short:   "Creates a default home directory at path defined by --home",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
				f, err := os.Create(cfgFile)
				if err != nil {
					return err
				}
				defer f.Close()
				if _, err = f.Write(defaultConfig()); err != nil {
					return err
				}
				fmt.Printf("Created config(%s)\n", cfgFile)
				return nil
			}
			return fmt.Errorf("config(%s) already exists", cfgFile)
		},
	}
	return cmd
}

func configDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete",
		Aliases: []string{"d"},
		Short:   "delete the config file at path --config",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if _, err = os.Stat(cfgFile); os.IsNotExist(err) {
				return fmt.Errorf("config(%s) doesn't exist", cfgFile)
			}
			if err = os.Remove(cfgFile); err != nil {
				return fmt.Errorf("error deleting config(%s)", err)
			}
			fmt.Printf("Removed config(%s)\n", cfgFile)
			return nil
		},
	}
	return cmd
}

func configShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show",
		Aliases: []string{"s"},
		Short:   "print existing config",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if _, err = os.Stat(cfgFile); os.IsNotExist(err) {
				return fmt.Errorf("config(%s) doesn't exist", cfgFile)
			}
			fmt.Print(string(config.MustYAML()))
			return nil
		},
	}
	return cmd
}

func configEditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "edit [key] [value]",
		Aliases: []string{"e"},
		Short:   "edit a given value in the config file",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			switch args[0] {
			case "rpc-addr":
				// TODO: validate
				config.RPCAddr = args[1]
				return overwriteConfig(cmd, config)
			case "chain-id":
				// TODO: validate
				config.ChainID = args[1]
				return overwriteConfig(cmd, config)
			case "build-repo":
				// TODO: validate
				config.BuildRepo = args[1]
				return overwriteConfig(cmd, config)
			case "build-command":
				// TODO: validate
				config.BuildCommand = args[1]
				return overwriteConfig(cmd, config)
			case "build-version":
				// TODO: validate
				config.BuildVersion = args[1]
				return overwriteConfig(cmd, config)
			case "binary-name":
				// TODO: validate
				config.BinaryName = args[1]
				return overwriteConfig(cmd, config)
			case "github-access-token":
				// TODO: validate
				config.GithubAccessToken = args[1]
				return overwriteConfig(cmd, config)
			case "registry-fork-name":
				// TODO: validate
				config.RegistryForkName = args[1]
				return overwriteConfig(cmd, config)
			case "registry-root-branch":
				// TODO: validate
				config.RegistryRootBranch = args[1]
				return overwriteConfig(cmd, config)
			case "git-name":
				// TODO: validate
				config.GitName = args[1]
				return overwriteConfig(cmd, config)
			case "git-email":
				// TODO: validate
				config.GitEmail = args[1]
				return overwriteConfig(cmd, config)
			case "commit-message":
				// TODO: validate
				config.CommitMessage = args[1]
				return overwriteConfig(cmd, config)
			default:
				return fmt.Errorf("key(%s) is not in the config file or is not editable via this command", args[0])
			}
		},
	}
	return cmd
}

func overwriteConfig(cmd *cobra.Command, cfg *registrar.Config) (err error) {
	if _, err = os.Stat(cfgFile); err == nil {
		out, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(cfgFile, out, 0600)
		if err != nil {
			return err
		}
		config = cfg
	}
	return nil
}
