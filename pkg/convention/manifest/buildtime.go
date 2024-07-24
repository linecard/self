package manifest

import (
	"path/filepath"

	"github.com/linecard/self/internal/gitlib"
)

type BuildTime struct {
	Release
	Path    string
	Context string
}

func Encode(path string, git gitlib.DotGit) (b BuildTime, err error) {
	s := Init()
	name := filepath.Base(path)

	if err = s.Schema.Encode(schemaVersion); err != nil {
		return b, err
	}

	if err = s.Name.Encode(name); err != nil {
		return b, err
	}

	if err = s.Branch.Encode(git.Branch); err != nil {
		return b, err
	}

	if err = s.Sha.Encode(git.Sha); err != nil {
		return b, err
	}

	if err = s.Origin.Encode(git.Origin.String()); err != nil {
		return b, err
	}

	if err = s.Role.Encode("embedded/roles/lambda.json.tmpl"); err != nil {
		return b, err
	}

	if err = s.Policy.Encode(filepath.Join(path, "policy.json.tmpl")); err != nil {
		return b, err
	}

	if err = s.Resources.Encode(filepath.Join(path, "resources.json.tmpl")); err != nil {
		return b, err
	}

	if err = s.Bus.Encode(filepath.Join(path, "bus")); err != nil {
		return b, err
	}

	return BuildTime{
		Path:    path,
		Context: path,
		Release: s,
	}, nil
}

func (b BuildTime) EncodedLabels() map[string]string {
	m := make(map[string]string)

	m[b.Schema.Key] = b.Schema.Encoded
	m[b.Name.Key] = b.Name.Encoded
	m[b.Branch.Key] = b.Branch.Encoded
	m[b.Sha.Key] = b.Sha.Encoded
	m[b.Origin.Key] = b.Origin.Encoded
	m[b.Role.Key] = b.Role.Encoded
	m[b.Policy.Key] = b.Policy.Encoded
	m[b.Resources.Key] = b.Resources.Encoded

	for _, bus := range b.Bus.Content {
		m[bus.Key] = bus.Encoded
	}

	return m
}
