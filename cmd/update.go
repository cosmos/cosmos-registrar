package cmd

import (
	"fmt"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	registrar "github.com/jackzampolin/cosmos-registrar/pkg/config"
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

	// build a list of the chainIDs that the user owns
	co, err := codeowners.FromFile(registryFolder)
	utils.AbortIfError(err, "cannot find the CODEOWNERS file: %v", err)
	chainIDs := myChains(co, config)

	var wg sync.WaitGroup
	// this are the actual nodes that we need to update
	for _, cID := range chainIDs {
		wg.Add(1)
		go func(rootFolder, chainID string) {
			defer wg.Done()
			peers, err := node.LoadPeers(rootFolder, chainID, config.RPCAddr, logger)
			utils.AbortIfError(err, "failed to load peer info for chain ID %s: %v", chainID, err)

			// contact all peers
			peersReachable := node.RefreshPeers(peers, logger)
			// save the changes
			node.SavePeers(rootFolder, chainID, peersReachable, logger)
			// commit and push
			err = gitwrap.StageToCommit(repo, chainID)
			utils.AbortIfError(err, "failed to stage updates to repository")
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

// myChains parses CODEOWNERS and returns a list of chainIDs that the
// config.GitName username owns
func myChains(co *codeowners.Codeowners, config *registrar.Config) (chainIDs []string) {
	chainIDs = []string{}
	if co.Patterns[0].Pattern == "*" && utils.ContainsStr(&co.Patterns[0].Owners, fmt.Sprint("@", config.GitName)) {
		logger.Info("you own everything", "pattern", co.Patterns[0].Pattern, "git username", config.GitName)
		for _, p := range co.Patterns[1:] {
			chainIDs = append(chainIDs, strings.Trim(p.Pattern, "/"))
		}
		return
	} else {
		for _, p := range co.Patterns {
			if utils.ContainsStr(&p.Owners, fmt.Sprint("@", config.GitName)) {
				logger.Info("found match", "pattern", p.Pattern, "git username", config.GitName)
				// TODO: validate this path before appending it (eg. doesn't contains special chars like ../ and so on)
				chainIDs = append(chainIDs, strings.Trim(p.Pattern, "/"))
			}
		}
	}
	return
}
