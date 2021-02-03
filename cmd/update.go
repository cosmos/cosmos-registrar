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
	"io/ioutil"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/jackzampolin/cosmos-registrar/pkg/node"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "updates the configured registry repo with the latest data",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var (
			repo     *git.Repository
			worktree *git.Worktree

			authOpts = &http.BasicAuth{
				Username: "emptystring",
				Password: config.GithubAccessToken,
			}
			cloneOpts = &git.CloneOptions{
				Auth:          authOpts,
				URL:           config.RegistryRoot,
				SingleBranch:  true,
				ReferenceName: plumbing.NewBranchReferenceName(config.RegistryRootBranch),
				Progress:      ioutil.Discard,
			}
			commitOpts = &git.CommitOptions{
				Author: &object.Signature{
					Name:  config.GitName,
					Email: config.GitEmail,
					When:  time.Now(),
				},
			}
			pushOpts = &git.PushOptions{
				Auth:     authOpts,
				Progress: ioutil.Discard,
			}
		)

		dir, err := ioutil.TempDir("", "registrar")
		if err != nil {
			return fmt.Errorf("creating temp directory: %s", err)
		}
		defer os.RemoveAll(dir)

		logger.Info("cloning", "registry-repo", config.RegistryRoot, "tmpdir", dir)
		if repo, err = git.PlainClone(dir, false, cloneOpts); err != nil {
			return fmt.Errorf("cloning %s repository: %s", config.RegistryRoot, err)
		}
		if worktree, err = repo.Worktree(); err != nil {
			return fmt.Errorf("creating git working tree: %s", err)
		}

		// dump node info
		chainID, err := node.FetchChainID(config)
		if err != nil {
			return err
		}
		err = node.DumpInfo(dir, chainID, config, logger)
		if err != nil {
			return err
		}
		println("updating ", chainID)

		st, err := worktree.Status()
		if err != nil {
			return err
		}
		for k := range st {
			if _, err = worktree.Add(k); err != nil {
				return err
			}
		}

		msg := fmt.Sprintf("Push master [%s]: %s", config.ChainID, config.CommitMessage)
		obj, err := worktree.Commit(msg, commitOpts)
		if err != nil {
			return fmt.Errorf("commiting changes on %s: %s", config.RegistryRoot, err)
		}
		if err = repo.Push(pushOpts); err != nil {
			return fmt.Errorf("pushing to %s: %s", config.RegistryRoot, err)
		}
		logger.Info("committed changes", "repo", config.RegistryRoot, "commit", obj.String(), "message", msg)
		return nil
	},
}
