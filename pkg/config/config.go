package config

import (
	"encoding/json"

	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"gopkg.in/yaml.v2"
)

// IsValid - check if the configuration is valid
func (c *Config) IsValid() bool {
	if len(c.GithubAccessToken) != 40 {
		return false
	}
	if len(c.GitName) == 0 {
		return false
	}
	if len(c.GitEmail) == 0 {
		return false
	}

	return true
}

// Config represents the configuration for the given application
type Config struct {
	RPCAddr            string `json:"rpc-addr" yaml:"-" mapstructure:"-"`
	ChainID            string `json:"chain-id" yaml:"-" mapstructure:"-"`
	BuildRepo          string `json:"build-repo" yaml:"-" mapstructure:"-"`
	BuildCommand       string `json:"build-command" yaml:"-" mapstructure:"-"`
	BinaryName         string `json:"binary-name" yaml:"-" mapstructure:"-"`
	BuildVersion       string `json:"build-version" yaml:"-" mapstructure:"-"`
	GithubAccessToken  string `json:"github-access-token" yaml:"github-access-token" mapstructure:"github-access-token"`
	RegistryRoot       string `json:"registry-root" yaml:"registry-root" mapstructure:"registry-root"`
	RegistryForkName   string `json:"registry-fork-name" yaml:"registry-fork-name" mapstructure:"registry-fork-name"`
	RegistryRootBranch string `json:"registry-root-branch" yaml:"registry-root-branch" mapstructure:"registry-root-branch"`
	GitName            string `json:"git-name" yaml:"git-name" mapstructure:"git-name"`
	GitEmail           string `json:"git-email" yaml:"git-email" mapstructure:"git-email"`
	CommitMessage      string `json:"commit-message" yaml:"-" mapstructure:"-"`
	// runtime variables
	Workspace string `json:"-" yaml:"-" mapstructure:"-"`
}

// Binary returns the binary file representation from the config
func (c *Config) Binary() []byte {
	// TODO: ensure this is sorted?
	out, _ := json.MarshalIndent(&Binary{
		Name:    c.BinaryName,
		Repo:    c.BuildRepo,
		Build:   c.BuildCommand,
		Version: c.BuildVersion,
	}, "", "  ")
	return out
}

// YAML converts the config into yaml bytes
func (c *Config) YAML() ([]byte, error) {
	return yaml.Marshal(c)
}

// MustYAML converts to yaml bytes panicing on error
func (c *Config) MustYAML() []byte {
	out, err := c.YAML()
	if err != nil {
		panic(err)
	}
	return out
}

// BasicAuth - build basic auth credentials from theconfiguration
func (c *Config) BasicAuth() *http.BasicAuth {
	return &http.BasicAuth{
		Username: c.GitName,
		Password: c.GithubAccessToken,
	}
}

// Binary is everything you need to build the binary
// for the network from the repo configured
type Binary struct {
	Name    string `json:"name"`
	Repo    string `json:"repo"`
	Build   string `json:"build"`
	Version string `json:"version"`
}
