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

type updates struct {
	lr    *node.LightRoot
	peers map[string]*node.Peer
}

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
		mu             sync.Mutex
		updatedInfo    = make(map[string]*updates)
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

	// update is meant to be called mostly by a CODEOWNER who owns the entire
	// repo. So one machine will contact all the chainIDs and push all the
	// updates. Contacting the chainIDs are done asynchronously
	var wg sync.WaitGroup
	for _, cID := range chainIDs {
		wg.Add(1)
		go func(rootFolder, chainID string) {
			defer wg.Done()
			peers, err := node.LoadPeers(rootFolder, chainID, config.RPCAddr, logger)
			if err != nil {
				logger.Error("failed to load peer info", "chainID", chainID, "err", err)
				return
			}

			// contact all peers, ask them for peers and check if those are up
			peersReachable := node.RefreshPeers(peers, logger)
			// ask reachable peers about light root hashes
			lr, err := node.UpdateLightRoots(chainID, peersReachable, logger)
			if err != nil {
				logger.Error("failed to update lightroots", "chainID", chainID, "err", err)
				return
			}

			u := &updates{
				lr:    lr,
				peers: peersReachable,
			}
			mu.Lock()
			updatedInfo[chainID] = u
			mu.Unlock()
		}(registryFolder, cID)
	}
	wg.Wait()

	// saving and committing the info is done synchronously.
	for chainID, u := range updatedInfo {
		// save the updated lightroot history
		err = node.SaveLightRoots(registryFolder, chainID, u.lr, logger)
		if err != nil {
			logger.Error("failed to save updated lightroots", "chainID", chainID, "err", err)
			return
		}
		// save the updated peerlist
		node.SavePeers(registryFolder, chainID, u.peers, logger)
		// commit and push
		err = gitwrap.StageToCommit(repo, chainID)
		if err != nil {
			logger.Error("failed to stage updates to repository", "chainID", chainID, "err", err)
			return
		}

		hash, err := gitwrap.Commit(repo,
			config.GitName,
			config.GitEmail,
			fmt.Sprintf("updates for chain id %s", chainID),
			time.Now(),
		)
		if err != nil {
			logger.Error("failed to commit updates to repository", "chainID", chainID, "err", err)
		} else {
			logger.Info("chain ID update committed", "chainID", chainID, "commitHash", hash)
		}
	}
	err = gitwrap.Push(repo, config.BasicAuth())
	utils.AbortIfError(err, "failed to update registry, please manually rollback the repo changes and try again")
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
