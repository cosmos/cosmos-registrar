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
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/spf13/cobra"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
	"golang.org/x/sync/errgroup"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "updates the registry repo with the latest data",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var (
			eg        errgroup.Group
			gen       *ctypes.ResultGenesis
			commit    *ctypes.ResultCommit
			netInfo   *ctypes.ResultNetInfo
			dir       string
			repo      *git.Repository
			worktree  *git.Worktree
			cloneOpts = &git.CloneOptions{
				URL:           config.RegistryRepo,
				SingleBranch:  true,
				ReferenceName: plumbing.NewBranchReferenceName(config.RegistryRepoBranch),
				// TODO: optionally log progress
				Progress: ioutil.Discard,
			}
			commitOpts = &git.CommitOptions{
				Author: &object.Signature{
					Name:  config.GitName,
					Email: config.GitEmail,
					When:  time.Now(),
				},
			}
		)

		// Fetch the tendermint client from the config
		clnt, err := config.Client()
		if err != nil {
			return err
		}

		// query the status from the chain
		// surface basic errors here
		stat, err := clnt.Status()
		switch {
		case err != nil:
			return err
		case stat.NodeInfo.Network != config.ChainID:
			return fmt.Errorf("node(%s) is on chain(%s) not configured chain(%s)", config.RPCAddr, stat.NodeInfo.Network, config.ChainID)
		case stat.SyncInfo.CatchingUp:
			return fmt.Errorf("node(%s) on chain(%s) still catching up", config.RPCAddr, config.ChainID)
		default:
		}

		// query genesis file from the chain
		eg.Go(func() error {
			gen, err = clnt.Genesis()
			return err
		})

		// query signed commit from the chain
		eg.Go(func() error {
			h := stat.SyncInfo.LatestBlockHeight
			commit, err = clnt.Commit(&h)
			return err
		})

		// query peer info from the chain
		eg.Go(func() error {
			netInfo, err = clnt.NetInfo()
			// TODO: in a more advanced version of this tool,
			// this would crawl the network a couple of hops
			// and find more peers
			return err
		})

		// wait for network operations to complete
		if err = eg.Wait(); err != nil {
			return err
		}

		// create temporary folder to work in
		if dir, err = ioutil.TempDir("", "registrar"); err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(dir)

		var (
			heightPath     string
			chainPath      = path.Join(dir, config.ChainID)
			genesisPath    = path.Join(chainPath, "genesis.json")
			genesisSumPath = path.Join(chainPath, "genesis.json.sum")
			lightRootPath  = path.Join(chainPath, "light-roots")
			latestPath     = path.Join(lightRootPath, "latest.json")
			binariesPath   = path.Join(chainPath, "binaries.json")
		)

		// clone the registry repo
		if repo, err = git.PlainClone(dir, false, cloneOpts); err != nil {
			return err
		}

		if worktree, err = repo.Worktree(); err != nil {
			return err
		}

		// ensure that chain directory exists, create it if it doesn't

		if _, err = os.Stat(chainPath); os.IsNotExist(err) {
			os.Mkdir(chainPath, os.ModePerm)
		}

		// ensure that genesis file exists, if it doesn't then write the one queried from the chain
		// TODO: maybe do sanity checks on the genesis file returned from the chain compared with the
		// one in the repo
		eg.Go(func() error {
			if _, err = os.Stat(genesisPath); os.IsNotExist(err) {
				sum, write, err := sortedGenesis(gen.Genesis)
				if err != nil {
					return err
				}
				if err = ioutil.WriteFile(genesisSumPath, []byte(sum), 0644); err != nil {
					return err
				}
				return ioutil.WriteFile(genesisPath, write, 0644)
			}
			return nil
		})

		// update light client files
		eg.Go(func() error {
			// ensure that light client root directory exists directory exists, create it if it doesn't
			if _, err = os.Stat(lightRootPath); os.IsNotExist(err) {
				os.Mkdir(lightRootPath, os.ModePerm)
			}

			// ensure that latest.json gets updated
			heightPath = path.Join(lightRootPath, fmt.Sprintf("%d.json", stat.SyncInfo.LatestBlockHeight))

			// we want to run these removes and ignore the errors
			os.Remove(latestPath)
			os.Remove(heightPath)

			// write the light root to latest
			if err = ioutil.WriteFile(latestPath, NewLightRoot(commit.SignedHeader), 0644); err != nil {
				return err
			}
			// and then write the height file
			return ioutil.WriteFile(heightPath, NewLightRoot(commit.SignedHeader), 0644)
		})

		// update binaries.json with current config
		eg.Go(func() error {
			// we want to run this remove and ignore the error
			os.Remove(binariesPath)
			// and write the info to the repo
			return ioutil.WriteFile(binariesPath, config.Binary(), 0644)
		})

		eg.Go(func() error {
			peersPath := path.Join(chainPath, "peers.json")
			var queryPeers = []string{}
			for _, p := range netInfo.Peers {
				queryPeers = append(queryPeers, fmt.Sprintf("%s@%s:%d", p.NodeInfo.ID(), p.RemoteIP, 26656))
			}
			// If there is no existing file, just dump the query peers to disk
			if _, err = os.Stat(peersPath); os.IsNotExist(err) {
				out, _ := json.MarshalIndent(queryPeers, "", "  ")
				return ioutil.WriteFile(peersPath, out, 0644)
			}

			// in the case where there are existing peers there, add any
			// new ones from the query
			var filePeers []string
			pf, err := os.Open(peersPath)
			if err != nil {
				return err
			}
			defer pf.Close()
			pfBytes, err := ioutil.ReadAll(pf)
			if err != nil {
				return err
			}
			if err = json.Unmarshal(pfBytes, &filePeers); err != nil {
				return err
			}
			// delete the file
			pf.Truncate(0)
			// marshal the deduped list
			toWrite, _ := json.MarshalIndent(dedupe(append(filePeers, queryPeers...)), "", "  ")
			// write it to the file
			_, err = pf.Write(toWrite)
			return err
		})

		// Wait for file operations
		if err = eg.Wait(); err != nil {
			return err
		}

		st, err := worktree.Status()
		if err != nil {
			return err
		}

		for k := range st {
			if _, err = worktree.Add(k); err != nil {
				return err
			}
		}

		// commit those changes
		commitMsg := fmt.Sprintf("Push master: bot update %s at %s", config.ChainID, time.Now())
		if _, err := worktree.Commit(commitMsg, commitOpts); err != nil {
			fmt.Println("in commit error")
			return err
		}

		// push those changes
		return repo.Push(&git.PushOptions{
			Auth: &http.BasicAuth{
				Username: "jackzampolin", // yes, this can be anything except an empty string
				Password: config.GithubAccessToken,
			},
			Progress: os.Stdout,
		})
	},
}

func sortedGenesis(gen *tmtypes.GenesisDoc) (sum string, indented []byte, err error) {
	// prepare to sort
	if indented, err = json.Marshal(gen); err != nil {
		return
	}

	// sort
	var c interface{}
	if err = json.Unmarshal(indented, &c); err != nil {
		return
	}

	// indent
	if indented, err = json.MarshalIndent(c, "", "  "); err != nil {
		return
	}

	// sum
	sum = fmt.Sprintf("%x", sha256.Sum256(indented))
	return
}

// Binary is everything you need to build the binary
// for the network from the repo configured
type Binary struct {
	Name    string `json:"name"`
	Repo    string `json:"repo"`
	Build   string `json:"build"`
	Version string `json:"version"`
}

// LightRoot is the format for a light client root file which
// will be used for state sync
type LightRoot struct {
	TrustHeight int64  `json:"trust-height"`
	TrustHash   string `json:"trust-hash"`
}

// NewLightRoot returns a new light root
func NewLightRoot(sh tmtypes.SignedHeader) []byte {
	out, _ := json.MarshalIndent(&LightRoot{
		TrustHeight: sh.Header.Height,
		TrustHash:   sh.Commit.BlockID.Hash.String(),
	}, "", "  ")
	return out
}

func dedupe(ele []string) (out []string) {
	e := map[string]bool{}
	for v := range ele {
		e[ele[v]] = true
	}
	for k := range e {
		out = append(out, k)
	}
	return
}
