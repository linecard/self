package gitlib

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type DotGit struct {
	// Path   string
	Branch string
	Sha    string
	Root   string
	Origin *url.URL
	Dirty  bool
}

func FromCwd() (found DotGit, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return DotGit{}, err
	}

	thisRepoPath, thisRepo, err := FindDotGit(cwd)
	if err != nil {
		return DotGit{}, err
	}

	if found.Branch, err = Branch(thisRepo); err != nil {
		return DotGit{}, err
	}

	if found.Sha, err = Sha(thisRepo); err != nil {
		return DotGit{}, err
	}

	if found.Origin, err = Origin(thisRepo); err != nil {
		return DotGit{}, err
	}

	if found.Dirty, err = Dirty(thisRepo); err != nil {
		return DotGit{}, err
	}

	// if found.Path, err = Path(thisRepo); err != nil {
	// 	return DotGit{}, err
	// }

	found.Root = thisRepoPath

	return found, nil
}

func FindDotGit(cwd string) (root string, repo *git.Repository, err error) {
	for {
		if _, err := os.Stat(filepath.Join(cwd, ".git")); err == nil {
			repo, err := git.PlainOpen(cwd)
			if err != nil {
				return "", nil, err
			}

			return cwd, repo, nil
		}

		parentDir := filepath.Dir(cwd)
		if parentDir == cwd {
			return cwd, nil, fmt.Errorf("this does not appear to be a git repository")
		}
		cwd = parentDir
	}
}

func Head(repo *git.Repository) (plumbing.Reference, error) {
	head, err := repo.Head()
	if err != nil {
		return plumbing.Reference{}, err
	}

	return *head, nil
}

// func Path(repo *git.Repository) (string, error) {
// 	origin, err := Origin(repo)
// 	if err != nil {
// 		return "", err
// 	}

// 	return strings.TrimSuffix(origin.Path, ".git"), nil
// }

func Branch(repo *git.Repository) (string, error) {
	head, err := Head(repo)
	if err != nil {
		return "", err
	}

	return head.Name().Short(), nil
}

func Sha(repo *git.Repository) (string, error) {
	head, err := Head(repo)
	if err != nil {
		return "", err
	}

	return head.Hash().String(), nil
}

func Dirty(repo *git.Repository) (bool, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return false, err
	}

	status, err := wt.Status()
	if err != nil {
		return false, err
	}

	return !status.IsClean(), nil
}

func Origin(repo *git.Repository) (*url.URL, error) {
	remote, err := repo.Remote("origin")
	if err != nil {
		return nil, err
	}

	urls := remote.Config().URLs
	if len(urls) == 0 {
		return nil, fmt.Errorf("no remote origin found")
	}

	if len(urls) > 1 {
		return nil, fmt.Errorf("multiple remote origins found")
	}

	if strings.HasPrefix(urls[0], "git@") {
		urls[0] = strings.Replace(urls[0], ":", "/", 1)
		urls[0] = strings.Replace(urls[0], "git@", "https://", 1)
	}

	return url.Parse(urls[0])
}
