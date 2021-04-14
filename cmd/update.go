package cmd

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/jackzampolin/cosmos-registrar/pkg/gitwrap"
	"github.com/jackzampolin/cosmos-registrar/pkg/node"
	"github.com/jackzampolin/cosmos-registrar/pkg/utils"
	"github.com/noandrea/go-codeowners"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "updates the configured registry repo with the latest data",
	RunE:  update,
}

func update(cmd *cobra.Command, args []string) (err error) {

	var (
		repo           *git.Repository
		registryFolder = path.Join(config.Workspace, "registry-root")
	)

	// open/clone and pull changes from the root repo
	_, err = url.Parse(config.RegistryRoot)
	utils.AbortIfError(err, "the registry root url is not a valid url: %s", config.RegistryRoot)

	repo, err = gitwrap.CloneOrOpen(config.RegistryRoot, registryFolder, config.BasicAuth())
	utils.AbortIfError(err, "aborted due to an error cloning registry fork repo: %v", err)

	err = gitwrap.PullBranch(repo, config.RegistryRootBranch)
	utils.AbortIfError(err, "error pulling changes for the %s branch: %v", config.RegistryRootBranch, err)

	// ###  list the entries that the user owns
	// read the the codeowners file
	co, err := codeowners.FromFile(registryFolder)
	utils.AbortIfError(err, "cannot find the CODEOWNERS file: %v", err)

	// TODO: move to config
	repoPatternRgxp := regexp.MustCompile("[a-zA-Z0-9]+(-[a-zA-Z0-9]+)*")
	// contains the path/chainIds to collect
	chainIDs := []string{}
	for _, p := range co.Patterns {
		if matched := repoPatternRgxp.MatchString(p.Pattern); !matched {
			logger.Error("Invalid path [SKIPPED]", "pattern", p)
			continue
		}
		if utils.ContainsStr(&p.Owners, config.GitName) {
			logger.Info("found path %s", p.Pattern)
			// TODO: validate this path before appending it (eg. doesn't contains special chars like ../ and so on)
			chainIDs = append(chainIDs, p.Pattern)
		}
	}

	wg := sync.WaitGroup{}
	// this are the actual nodes that we need to update
	for _, cID := range chainIDs {
		go func(rootFolder, chainID string) {
			defer wg.Done()
			peers, err := node.LoadPeers(rootFolder, chainID, config.RPCAddr, logger)
			utils.AbortIfError(err, "failed to load peer info for chain ID %s: %v", chainID, err)
			// load genesis checksum
			checksum, err := node.LoadGenesisSum(rootFolder, chainID)
			if err != nil {
				logger.Error("permanent error: cannot retrieve the genesis checksum", "repo", rootFolder, "chain ID", chainID)
				return
			}
			// contact all peers
			node.RefreshPeers(peers, checksum, logger)
			// save the changes
			node.SavePeers(rootFolder, chainID, peers, logger)
			// commit and push
			// now commit the changes
			hash, err := gitwrap.CommitAndPush(repo,
				config.GitName,
				config.GitEmail,
				fmt.Sprintf("updates for chain id %s", chainID),
				time.Now(),
				config.BasicAuth(),
			)
			utils.AbortIfError(err, "failed to update registry, please manually rollback the repo changes and try again")
			logger.Info("chain ID update committed", "chainID", chainID, "commitHash", hash)
		}(registryFolder, cID)
	}
	wg.Wait()

	return
}
