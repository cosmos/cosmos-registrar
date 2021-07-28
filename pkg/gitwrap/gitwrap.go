package gitwrap

import (
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp/sideband"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/muja/goconfig"
	"github.com/spf13/afero"
)

var (
	// ProgressOutout control git command output
	ProgressOutout sideband.Progress = ioutil.Discard
	fs                               = afero.NewOsFs()
)

// GetGlobalGitIdentity - attempt to read git identity from global config
func GetGlobalGitIdentity() (user, email string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	fs := afero.NewOsFs()
	gitconfig := path.Join(home, ".gitconfig")
	if xs, err := afero.Exists(fs, gitconfig); err != nil || !xs {
		return
	}
	data, err := afero.ReadFile(fs, gitconfig)
	if err != nil {
		return
	}
	goconfig.Parse(data)
	config, _, err := goconfig.Parse(data)
	if err != nil {
		return
	}
	user = config["user.username"]
	email = config["user.email"]
	return
}

// CloneOrOpen - shortcut to cloning a repo or opening an existing one
func CloneOrOpen(repoURL, destFolder string, auth *http.BasicAuth) (repo *git.Repository, err error) {
	forkExists, err := afero.DirExists(fs, destFolder)
	if err != nil {
		return
	}
	if !forkExists {
		println("cloning the root registry from", repoURL)
		// create the workspace just in case
		err = fs.MkdirAll(destFolder, 0700)
		if err != nil {
			return
		}

		// clone the repo
		repo, err = git.PlainClone(destFolder, false, &git.CloneOptions{
			URL:      repoURL,
			Progress: ProgressOutout,
			Auth:     auth,
		})
		return
	}
	repo, err = git.PlainOpen(destFolder)
	return
}

// PullBranch - checkout + fetch + merge an existing branch
func PullBranch(repo *git.Repository, branchName string) (err error) {
	wt, err := repo.Worktree()
	if err != nil {
		return
	}
	// move to main branch
	// TODO: what if the path isn't clean?
	err = wt.Checkout(&git.CheckoutOptions{
		Create: false,
		Branch: plumbing.NewBranchReferenceName(branchName),
	})
	if err != nil {
		return
	}
	err = wt.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != nil {
		if err == git.NoErrAlreadyUpToDate {
			return nil
		}
	}
	return
}

// CreateBranch - create a new branch
func CreateBranch(repo *git.Repository, branchName string) (err error) {
	wt, err := repo.Worktree()
	if err != nil {
		return
	}
	err = wt.Checkout(&git.CheckoutOptions{
		Create: true,
		Branch: plumbing.NewBranchReferenceName(branchName),
	})
	return
}

// StageToCommit - stage paths to be committed
func StageToCommit(repo *git.Repository, path ...string) (err error) {
	wt, err := repo.Worktree()
	if err != nil {
		return
	}
	for _, p := range path {
		_, err = wt.Add(p)
		if err != nil {
			return err
		}
	}
	return
}

// Commit forms a commit from already staged files
func Commit(repo *git.Repository, name, email, message string, date time.Time) (hash string, err error) {
	wt, err := repo.Worktree()
	if err != nil {
		return
	}
	commit, err := wt.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  name,
			Email: email,
			When:  date,
		},
	})
	if err != nil {
		// TODO is it a good idea to reset?
		wt.Reset(&git.ResetOptions{})
		return
	}
	hash = commit.String()
	return hash, nil
}

func Push(repo *git.Repository, auth *http.BasicAuth) (err error) {
	wt, err := repo.Worktree()
	if err != nil {
		return
	}

	for retries := 0; retries < 10; retries++ {
		// pull first
		err = wt.Pull(&git.PullOptions{RemoteName: "origin"})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return
		}
		err = repo.Push(&git.PushOptions{
			Auth:     auth,
			Progress: ProgressOutout,
		})

		switch err {
		case nil, git.NoErrAlreadyUpToDate:
			return
		}
	}

	return
}

// CommitAndPush - shortcut for commit and push
// name, email, date are for the user creating the commit singnature
// message is the commit message
// auth is for authentication to the remote for the pull
func CommitAndPush(repo *git.Repository, name, email, message string, date time.Time, auth *http.BasicAuth) (hash string, err error) {
	hash, err = Commit(repo, name, email, message, date)
	if err != nil {
		return
	}

	err = Push(repo, auth)
	return
}
