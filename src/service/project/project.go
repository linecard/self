package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type Git struct {
	Branch string
	Sha    string
	Root   string
}

func FromCwd() (Git, error) {
	branch, err := Branch()
	if err != nil {
		return Git{}, err
	}

	sha, err := Sha()
	if err != nil {
		return Git{}, err
	}

	root, err := Root()
	if err != nil {
		return Git{}, err
	}

	return Git{branch, sha, root}, nil
}

func Head() (plumbing.Reference, error) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return plumbing.Reference{}, err
	}

	head, err := repo.Head()
	if err != nil {
		return plumbing.Reference{}, err
	}

	return *head, nil
}

func Branch() (string, error) {
	head, err := Head()
	if err != nil {
		return "", err
	}

	return head.Name().Short(), nil
}

func Sha() (string, error) {
	head, err := Head()
	if err != nil {
		return "", err
	}

	return head.Hash().String(), nil
}

func Root() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}

		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			return "", fmt.Errorf("no Git repository found")
		}
		dir = parentDir
	}
}
