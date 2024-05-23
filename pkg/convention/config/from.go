package config

import (
	"path/filepath"
	"strings"

	"github.com/linecard/self/internal/umwelt"
)

func FromHere(here umwelt.Here) (c Config) {
	c.Function = (*Function)(here.Function)

	for _, f := range here.Functions {
		c.Functions = append(c.Functions, (Function)(f))
	}

	c.Caller.Arn = here.Caller.Arn

	c.Account.Id = here.Caller.Account
	c.Account.Region = here.Caller.Region

	c.Registry.Id = here.Registry.Id
	c.Registry.Region = here.Registry.Region
	c.Registry.Url = c.Registry.Id + ".dkr.ecr." + c.Registry.Region + ".amazonaws.com"

	c.Git.Origin = here.Git.Origin.String()
	c.Git.Branch = here.Git.Branch
	c.Git.Sha = here.Git.Sha
	c.Git.Root = here.Git.Root
	c.Git.Dirty = here.Git.Dirty

	c.Repository.Prefix = strings.Replace(strings.TrimPrefix(here.Git.Origin.Path, "/"), ".git", "", 1)

	c.Resource.Prefix = strings.Replace(filepath.Base(here.Git.Origin.Path), ".git", "", 1)

	c.TemplateData.AccountId = c.Account.Id
	c.TemplateData.Region = c.Account.Region
	c.TemplateData.RegistryRegion = c.Registry.Region
	c.TemplateData.RegistryAccountId = c.Registry.Id

	c.Label.Role = "org.linecard.self.role"
	c.Label.Policy = "org.linecard.self.policy"
	c.Label.Sha = "org.linecard.self.git-sha"
	c.Label.Bus = "org.linecard.self.bus"
	c.Label.Resources = "org.linecard.self.resources"

	c.Httproxy.ApiId = here.ApiGateway.Id

	c.Version = version

	return
}
