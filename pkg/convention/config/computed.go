package config

import (
	"encoding/json"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/linecard/self/internal/gitlib"
	"github.com/linecard/self/internal/util"
	"github.com/linecard/self/pkg/convention/manifest"
)

type Repository struct {
	Prefix string
	Name   string
	Path   string
	Url    string
}

type Resource struct {
	Prefix string
	Name   string
	Policy struct {
		Arn string
	}
	Role struct {
		Arn string
	}
}

type Resources struct {
	EphemeralStorage int32  `json:"ephemeralStorage"`
	MemorySize       int32  `json:"memorySize"`
	Timeout          int32  `json:"timeout"`
	Http             bool   `json:"http"`
	Public           bool   `json:"public"`
	RouteKey         string `json:"routeKey"`
}

type BuildTimeDerivation struct {
	Registry   Registry
	Repository Repository
}

type DeployTimeDerivation struct {
	Registry   Registry
	Repository Repository
	Resource   Resource
	Resources  Resources
}

func (c Config) DeriveBuildTime(mfst *manifest.BuildTime) (d BuildTimeDerivation, err error) {
	d.Registry.Url = c.Registry.Url
	d.Repository.Prefix = strings.TrimSuffix(c.Git.Origin.Path, ".git")
	d.Repository.Name = strings.TrimPrefix(d.Repository.Prefix, "/") + "/" + mfst.Name.Raw
	d.Repository.Path = filepath.Clean(d.Repository.Prefix + "/" + mfst.Name.Raw)
	d.Repository.Url = d.Registry.Url + d.Repository.Path
	return d, nil
}

func (c Config) DeriveDeployTime(mfst *manifest.DeployTime, imageUri string) (d DeployTimeDerivation, err error) {
	origin, err := url.Parse(mfst.Origin.Content)
	if err != nil {
		return d, err
	}

	git := gitlib.DotGit{
		Branch: mfst.Branch.Content,
		Sha:    mfst.Sha.Content,
		Origin: origin,
	}

	d.Registry.Url = strings.Split(imageUri, "@")[0]
	d.Repository.Prefix = strings.TrimSuffix(git.Origin.Path, ".git")
	d.Repository.Name = strings.TrimPrefix(d.Repository.Prefix, "/") + "/" + mfst.Name.Content
	d.Repository.Path = filepath.Clean(d.Repository.Prefix + "/" + mfst.Name.Content)
	d.Repository.Url = d.Registry.Url + d.Repository.Path
	d.Resource.Prefix = util.DeSlasher(d.Repository.Prefix) + "-" + util.DeSlasher(mfst.Branch.Content)
	d.Resource.Name = d.Resource.Prefix + "-" + mfst.Name.Content
	d.Resource.Policy.Arn = "arn:aws:iam::" + c.Account.Id + ":policy/" + d.Resource.Name
	d.Resource.Role.Arn = "arn:aws:iam::" + c.Account.Id + ":role/" + d.Resource.Name

	// merge defaults with given.
	resources := Resources{
		EphemeralStorage: 512,
		MemorySize:       128,
		Timeout:          3,
		Http:             true,
		Public:           false,
		RouteKey:         "ANY /" + d.Repository.Prefix + "/" + git.Branch + "/" + mfst.Name.Content,
	}

	if mfst.Resources.Content != "" {
		if err = json.Unmarshal([]byte(mfst.Resources.Content), &resources); err != nil {
			d.Resources = resources
		}
	} else {
		d.Resources = resources
	}

	return d, nil
}
