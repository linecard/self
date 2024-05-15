package mocks

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"

	mockfixture "github.com/linecard/self/mock/fixture"

	"github.com/linecard/self/internal/gitlib"
)

func MockRepository(orgName, repoName, branchName string, functionNames ...string) (gitMock gitlib.DotGit, cleanupHook func()) {
	for _, function := range functionNames {
		basePath := filepath.Join(repoName, function)
		srcPath := filepath.Join(basePath, "src")
		busPath := filepath.Join(basePath, "bus", "default")

		// Create Directories
		if err := os.MkdirAll(srcPath, os.ModePerm); err != nil {
			log.Fatal(err)
		}
		if err := os.MkdirAll(busPath, os.ModePerm); err != nil {
			log.Fatal(err)
		}

		// Copy Fixtures
		policyDst := filepath.Join(basePath, "policy.json.tmpl")
		if err := mockfixture.Copy("policy.json.tmpl", policyDst); err != nil {
			log.Fatal(err)
		}

		busDst := filepath.Join(busPath, "bus.json.tmpl")
		if err := mockfixture.Copy("bus.json.tmpl", busDst); err != nil {
			log.Fatal(err)
		}

		dockerfilePath := filepath.Join(basePath, "Dockerfile")
		if _, err := os.Create(dockerfilePath); err != nil {
			log.Fatal(err)
		}
	}

	mockGit := mockGit(orgName, repoName, branchName)

	cleanupHook = func() {
		os.RemoveAll(repoName)
	}

	return mockGit, cleanupHook
}

func mockGit(org, path, branch string) gitlib.DotGit {
	origin, err := url.Parse("https://github.com/" + org + "/" + path + ".git")
	if err != nil {
		log.Fatal(err)
	}

	sha, err := shaPath(path)
	if err != nil {
		log.Fatal(err)
	}

	return gitlib.DotGit{
		Branch: branch,
		Sha:    sha,
		Root:   path,
		Origin: origin,
		Dirty:  false,
	}
}

func shaPath(path string) (string, error) {
	hasher := sha1.New()

	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		_, err = hasher.Write([]byte(p))
		if err != nil {
			return err
		}

		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(hasher, f); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
