package manifest

import (
	"embed"
)

//go:embed embedded/*
var embedded embed.FS

const (
	schemaVersion = "1.1"
)

type Release struct {
	Schema    StringLabel
	Name      StringLabel
	Branch    StringLabel
	Sha       StringLabel
	Origin    StringLabel
	Role      EmbeddedFileLabel
	Policy    FileLabel
	Resources FileLabel
	Bus       FolderLabel
}

func Init() Release {
	return Release{
		Schema: StringLabel{
			Description: "Manifest schema version string",
			Key:         "org.linecard.self.schema",
			Required:    true,
		},
		Name: StringLabel{
			Description: "Function name string",
			Key:         "org.linecard.self.name",
			Required:    true,
		},
		Branch: StringLabel{
			Description: "Git branch string",
			Key:         "org.linecard.self.git.branch",
			Required:    true,
		},
		Sha: StringLabel{
			Description: "Git sha string",
			Key:         "org.linecard.self.git.sha",
			Required:    true,
		},
		Origin: StringLabel{
			Description: "Git origin string",
			Key:         "org.linecard.self.git.origin",
			Required:    true,
		},
		Role: EmbeddedFileLabel{
			Description: "Role template file",
			Key:         "org.linecard.self.role",
			Required:    true,
		},
		Policy: FileLabel{
			Description: "Policy template file",
			Key:         "org.linecard.self.policy",
			Required:    true,
		},
		Resources: FileLabel{
			Description: "Resources template file",
			Key:         "org.linecard.self.resources",
			Required:    false,
		},
		Bus: FolderLabel{
			Description: "Bus templates path",
			KeyPrefix:   "org.linecard.self.bus",
			Required:    false,
		},
	}
}
