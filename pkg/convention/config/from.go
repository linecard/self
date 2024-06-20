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

	c.Label.Sha = "org.linecard.self.git-sha"
	c.Label.Role = "org.linecard.self.role"
	c.Label.Policy = "org.linecard.self.policy"
	c.Label.Resources = "org.linecard.self.resources"
	c.Label.Bus = "org.linecard.self.bus"

	c.Labels.Schema = StringLabel{
		Description: "Label schema version string",
		Key:         "org.linecard.self.schema",
		Content:     "1.0",
	}

	c.Labels.Sha = StringLabel{
		Description: "Git sha string",
		Key:         "org.linecard.self.git-sha",
		Content:     c.Git.Sha,
	}

	c.Labels.Role = EmbeddedFileLabel{
		Description: "Role template file",
		Key:         "org.linecard.self.role",
		Path:        "embedded/roles/lambda.json.tmpl",
		Required:    true,
	}

	c.Labels.Policy = FileLabel{
		Description: "Policy template file",
		Key:         "org.linecard.self.policy",
		Path:        filepath.Join(c.Function.Path, "policy.json.tmpl"),
		Required:    true,
	}

	c.Labels.Resources = FileLabel{
		Description: "Resources template file",
		Key:         "org.linecard.self.resources",
		Path:        filepath.Join(c.Function.Path, "resources.json.tmpl"),
	}

	c.Labels.Bus = FolderLabel{
		Description: "Bus templates folder",
		KeyPrefix:   "org.linecard.self.bus",
		Path:        filepath.Join(c.Function.Path, "bus"),
	}

	c.Httproxy.ApiId = here.ApiGateway.Id

	return
}
