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
		},
		Name: StringLabel{
			Description: "Function name string",
			Key:         "org.linecard.self.name",
		},
		Branch: StringLabel{
			Description: "Git branch string",
			Key:         "org.linecard.self.git.branch",
		},
		Sha: StringLabel{
			Description: "Git sha string",
			Key:         "org.linecard.self.git.sha",
		},
		Origin: StringLabel{
			Description: "Git origin string",
			Key:         "org.linecard.self.git.origin",
		},
		Role: EmbeddedFileLabel{
			Description: "Role template file",
			Key:         "org.linecard.self.role",
		},
		Policy: FileLabel{
			Description: "Policy template file",
			Key:         "org.linecard.self.policy",
		},
		Resources: FileLabel{
			Description: "Resources template file",
			Key:         "org.linecard.self.resources",
		},
		Bus: FolderLabel{
			Description: "Bus templates path",
			KeyPrefix:   "org.linecard.self.bus",
		},
	}
}
