package cmd

import (
	"fmt"
	"os"
	"path"

	registrar "github.com/jackzampolin/cosmos-registrar/pkg/config"
	"github.com/jackzampolin/cosmos-registrar/pkg/gitwrap"
	"github.com/jackzampolin/cosmos-registrar/pkg/prompts"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/libs/log"
)

func init() {
	// Set Defaults
	// viper.SetDefault("rpc-addr", "http://localhost:26657")
	// viper.SetDefault("chain-id", "cosmoshub-3")

	// TODO how to get this data? can it be retrieved or shall be asked to the user?
	// Asking to the user may result in not being a reliable info
	// viper.SetDefault("build-repo", "https://github.com/cosmos/gaia")
	// viper.SetDefault("build-command", "gaiad")
	// viper.SetDefault("binary-name", "make install")
	// viper.SetDefault("build-version", "v2.0.13")

	viper.SetDefault("github-access-token", "get yours at https://github.com/settings/tokens")
	viper.SetDefault("registry-root", "https://github.com/apeunit/registry")
	viper.SetDefault("registry-fork-name", "registry")
	viper.SetDefault("registry-root-branch", "main")
	viper.SetDefault("git-name", "Your name goes here")
	viper.SetDefault("git-email", "your@email.here")
	// viper.SetDefault("commit-message", "update roots of trust")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if !debug {
		log.AllowLevel("debug")
	}

	config = &registrar.Config{}
	configPath, err := os.UserConfigDir()
	if err != nil {
		// TODO handle this
		panic("cannot retrieve the user configuration directory")
	}
	defaultCfgPath := path.Join(configPath, "cosmos", "registry")
	afero.NewOsFs().MkdirAll(defaultCfgPath, 0700)

	viper.SetConfigType("yaml")

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(defaultCfgPath)
		viper.SetConfigName("config")
	}

	// set the workspace folder
	cfgFilePath := viper.ConfigFileUsed()
	if cfgFilePath == "" {
		cfgFilePath = path.Join(defaultCfgPath, "config.yaml")
	}
	//now read in the config
	if err := viper.ReadInConfig(); err == nil {
		viper.Unmarshal(config)
		logger.Debug("Using config file at ", "config", viper.ConfigFileUsed())
	} else {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			if noInteraction {
				println("config file not found")
				os.Exit(1)
			}
			if err = interactiveSetup(); err != nil {
				println("unexpected error ", err.Error())
				os.Exit(1)
			}
			viper.Unmarshal(config)
			println("\nThe configuration is:")
			prompts.PrettyMap(viper.AllSettings())
			println()
			if ok := prompts.Confirm(false, "save the configuration?"); !ok {
				println("aborting, run the command again to change the configuration")
				os.Exit(0)
			}

			if err = viper.WriteConfigAs(cfgFilePath); err != nil {
				println("aborting, error writing the configuration file:", err.Error())
				os.Exit(1)
			}
			println("config file saved in ", cfgFilePath)

		default:
			println("the configuration file appers corrupted: ", err.Error())
			os.Exit(1)
		}
	}
	// set the config workspace folder
	config.Workspace = path.Dir(cfgFilePath)
}

// Setup guides the user to setup their environmant
func interactiveSetup() (err error) {
	println(`
Welcome to the Cosmos registry tool. 
This tool will allow you to publicly claim a Chain ID 
for your cosmos based chain.

To complete the setup you need a GitHub account and 
network connectivity to a node of your chain.
`)
	// ask to start
	if goOn := prompts.Confirm(true, "do you have them available"); !goOn {
		println("please make sure you get them and the run the setup again")
		os.Exit(0)
	}

	user, email := gitwrap.GetGlobalGitIdentity()

	// next get the git user
	gitName, err := prompts.InputOrDefault(user, "enter your git username")
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
	viper.Set("git-name", gitName)

	// next get the git email
	gitEmail, err := prompts.InputOrDefault(email, "enter your git email")
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
	viper.Set("git-email", gitEmail)

	// now get the github token
	println(`
The next step is to enter a github personal token for your account , 
if you don't have one you can get it from

https://github.com/settings/tokens

(make sure that you select the permission repo > public_repo)
`)

	token, err := prompts.Password("%s personal token", gitName)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
	viper.Set("github-access-token", token)

	println("the setup is now completed")
	return
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

// Command for initializing an empty config at the --home location
func configInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "init",
		Aliases: []string{"i"},
		Short:   "Creates a default home directory at path defined by --home",
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			err = viper.SafeWriteConfig()
			if err != nil {
				fmt.Printf("error saving config(%s): %v", viper.ConfigFileUsed(), err)
			}
			return

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
			if viper.ConfigFileUsed() == "" {
				return fmt.Errorf("config doesn't exist")
			}
			if _, err = os.Stat(viper.ConfigFileUsed()); os.IsNotExist(err) {
				return fmt.Errorf("config(%s) doesn't exist", cfgFile)
			}
			if err = os.Remove(viper.ConfigFileUsed()); err != nil {
				return fmt.Errorf("error deleting config(%s)", err)
			}
			fmt.Printf("Removed config(%s)\n", cfgFile)
			return
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
			if viper.ConfigFileUsed() == "" {
				return fmt.Errorf("config doesn't exist")
			}
			prompts.PrettyMap(viper.AllSettings())
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
	return viper.WriteConfig()
}
