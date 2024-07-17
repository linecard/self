package config

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type ReleaseSchema struct {
	Path      string
	Schema    StringLabel
	Name      StringLabel
	Branch    StringLabel
	Sha       StringLabel
	Origin    StringLabel
	Role      EmbeddedFileLabel
	Policy    FileLabel
	Resources FileLabel
	Bus       FolderLabel
	Computed  Computed
}

type BuildTime struct {
	Path      string
	Context   string
	Schema    EncodedLabel
	Name      EncodedLabel
	Branch    EncodedLabel
	Sha       EncodedLabel
	Origin    EncodedLabel
	Role      EncodedLabel
	Policy    EncodedLabel
	Resources EncodedLabel
	Bus       []EncodedLabel
	Computed  BuildTimeComputed
}

type DeployTime struct {
	Schema    DecodedLabel
	Name      DecodedLabel
	Branch    DecodedLabel
	Sha       DecodedLabel
	Origin    DecodedLabel
	Role      DecodedLabel
	Policy    DecodedLabel
	Resources DecodedLabel
	Bus       []DecodedLabel
	Computed  DeployTimeComputed
}

type ComputedPolicy struct {
	Arn  string
	Name string
}

type ComputedRole struct {
	Arn  string
	Name string
}

type ComputedResource struct {
	Prefix string
	Name   string
}

type ComputedResources struct {
	EphemeralStorage int32  `json:"ephemeralStorage"`
	MemorySize       int32  `json:"memorySize"`
	Timeout          int32  `json:"timeout"`
	Http             bool   `json:"http"`
	Public           bool   `json:"public"`
	RouteKey         string `json:"routeKey"`
}

type ComputedRepository struct {
	Prefix string
	Name   string
	Url    string
}

type ComputedImage struct {
	Uri  string
	Arch types.Architecture
}

type ComputedRegistry struct {
	Url string
}

type Computed struct {
	Registry   ComputedRegistry
	Repository ComputedRepository
	Resource   ComputedResource
}

type BuildTimeComputed struct {
	Registry   ComputedRegistry
	Repository ComputedRepository
}

type DeployTimeComputed struct {
	Registry   ComputedRegistry
	Repository ComputedRepository
	Resource   ComputedResource
	Resources  ComputedResources // find better name
	Image      ComputedImage
	Policy     ComputedPolicy
	Role       ComputedRole
}

func (r *ReleaseSchema) Encode(c Config) (b BuildTime, err error) {
	b.Path = r.Path
	b.Context = r.Path

	// Derive properties before encoding schema to base64
	b.Computed.Registry.Url = c.Registry.Id + ".dkr.ecr." + c.Registry.Region + ".amazonaws.com"
	b.Computed.Repository.Prefix = strings.Replace(filepath.Base(r.Origin.Content), ".git", "", 1)
	b.Computed.Repository.Name = b.Computed.Repository.Prefix + "/" + r.Name.Content
	b.Computed.Repository.Url = b.Computed.Registry.Url + "/" + b.Computed.Repository.Name

	if b.Schema, err = r.Schema.Encode(); err != nil {
		return b, err
	}

	if b.Name, err = r.Name.Encode(); err != nil {
		return b, err
	}

	if b.Branch, err = r.Branch.Encode(); err != nil {
		return b, err
	}

	if b.Sha, err = r.Sha.Encode(); err != nil {
		return b, err
	}

	if b.Origin, err = r.Origin.Encode(); err != nil {
		return b, err
	}

	if b.Role, err = r.Role.Encode(); err != nil {
		return b, err
	}

	if b.Policy, err = r.Policy.Encode(); err != nil {
		return b, err
	}

	if b.Resources, err = r.Resources.Encode(); err != nil {
		return b, err
	}

	if b.Bus, err = r.Bus.Encode(); err != nil {
		return b, err
	}

	return b, nil
}

func (b *BuildTime) LabelMap() map[string]string {
	m := make(map[string]string)

	m[b.Schema.Key] = b.Schema.Value
	m[b.Name.Key] = b.Name.Value
	m[b.Branch.Key] = b.Branch.Value
	m[b.Sha.Key] = b.Sha.Value
	m[b.Origin.Key] = b.Origin.Value
	m[b.Role.Key] = b.Role.Value
	m[b.Policy.Key] = b.Policy.Value
	m[b.Resources.Key] = b.Resources.Value

	for _, bus := range b.Bus {
		m[bus.Key] = bus.Value
	}

	return m
}

func (r ReleaseSchema) Decode(accountId, registryId, registryRegion string, labels map[string]string) (d DeployTime, err error) {
	if d.Schema, err = r.Schema.Decode(labels); err != nil {
		return d, err
	}

	if d.Name, err = r.Name.Decode(labels); err != nil {
		return d, err
	}

	if d.Branch, err = r.Branch.Decode(labels); err != nil {
		return d, err
	}

	if d.Sha, err = r.Sha.Decode(labels); err != nil {
		return d, err
	}

	if d.Origin, err = r.Origin.Decode(labels); err != nil {
		return d, err
	}

	if d.Role, err = r.Role.Decode(labels); err != nil {
		return d, err
	}

	if d.Policy, err = r.Policy.Decode(labels); err != nil {
		return d, err
	}

	if d.Resources, err = r.Resources.Decode(labels); err != nil {
		return d, err
	}

	if d.Bus, err = r.Bus.Decode(labels); err != nil {
		return d, err
	}

	// Derive properties after decoding schema from base64
	d.Computed.Registry.Url = r.Computed.Registry.Url

	d.Computed.Resource.Prefix = r.Computed.Resource.Prefix
	d.Computed.Resource.Name = r.Computed.Resource.Name

	d.Computed.Repository.Prefix = r.Computed.Repository.Prefix
	d.Computed.Repository.Name = r.Computed.Repository.Name
	d.Computed.Repository.Url = r.Computed.Repository.Url

	d.Computed.Policy.Name = r.Computed.Resource.Name
	d.Computed.Policy.Arn = "arn:aws:iam::" + accountId + ":policy/" + r.Computed.Resource.Name

	d.Computed.Role.Name = r.Computed.Resource.Name
	d.Computed.Role.Arn = "arn:aws:iam::" + accountId + ":role/" + r.Computed.Resource.Name

	resources := ComputedResources{
		EphemeralStorage: 512,
		MemorySize:       128,
		Timeout:          3,
		Http:             true,
		Public:           false,
		RouteKey:         "ANY /" + r.Computed.Resource.Prefix + "/" + d.Branch.Value + "/" + d.Name.Value,
	}

	if err = json.Unmarshal([]byte(d.Resources.Value), &resources); err != nil {
		return d, err
	}
	d.Computed.Resources = resources

	return d, nil
}

func (d *DeployTime) Template(data TemplateData) (err error) {
	if d.Schema.Value, err = templateString(d.Schema.Value, data); err != nil {
		return err
	}

	if d.Name.Value, err = templateString(d.Name.Value, data); err != nil {
		return err
	}

	if d.Branch.Value, err = templateString(d.Branch.Value, data); err != nil {
		return err
	}

	if d.Sha.Value, err = templateString(d.Sha.Value, data); err != nil {
		return err
	}

	if d.Origin.Value, err = templateString(d.Origin.Value, data); err != nil {
		return err
	}

	if d.Role.Value, err = templateString(d.Role.Value, data); err != nil {
		return err
	}

	if d.Policy.Value, err = templateString(d.Policy.Value, data); err != nil {
		return err
	}

	if d.Resources.Value, err = templateString(d.Resources.Value, data); err != nil {
		return err
	}

	for i, bus := range d.Bus {
		if d.Bus[i].Value, err = templateString(bus.Value, data); err != nil {
			return err
		}
	}

	return nil
}
