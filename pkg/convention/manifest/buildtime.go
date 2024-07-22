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

func (r *Release) Encode(path string, git gitlib.DotGit) (b BuildTime, err error) {
	name := filepath.Base(path)

	if err = r.Schema.Encode(schemaVersion); err != nil {
		return b, err
	}

	if err = r.Name.Encode(name); err != nil {
		return b, err
	}

	if err = r.Branch.Encode(git.Branch); err != nil {
		return b, err
	}

	if err = r.Sha.Encode(git.Sha); err != nil {
		return b, err
	}

	if err = r.Origin.Encode(git.Origin.String()); err != nil {
		return b, err
	}

	if err = r.Role.Encode("roles/lambda.json.tmpl"); err != nil {
		return b, err
	}

	if err = r.Policy.Encode(filepath.Join(path, "policy.json.tmpl")); err != nil {
		return b, err
	}

	if err = r.Resources.Encode(filepath.Join(path, "resources.json.tmpl")); err != nil {
		return b, err
	}

	if err = r.Bus.Encode(filepath.Join(path, "bus")); err != nil {
		return b, err
	}

	return BuildTime{
		Path:    path,
		Context: git.Root,
		Release: *r,
	}, nil
}

func (b BuildTime) LabelMap() map[string]string {
	m := make(map[string]string)

	m[b.Schema.Key] = b.Schema.Content
	m[b.Name.Key] = b.Name.Content
	m[b.Branch.Key] = b.Branch.Content
	m[b.Sha.Key] = b.Sha.Content
	m[b.Origin.Key] = b.Origin.Content
	m[b.Role.Key] = b.Role.Content
	m[b.Policy.Key] = b.Policy.Content
	m[b.Resources.Key] = b.Resources.Content

	for _, bus := range b.Bus.Content {
		m[bus.Key] = bus.Content
	}

	return m
}
