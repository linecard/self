package config

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/linecard/self/internal/util"
)

//go:embed embedded/*
var embedded embed.FS

type Function struct {
	Name string
	Path string
}

type Caller struct {
	Arn string
}

type Account struct {
	Id     string
	Region string
}

type Git struct {
	Origin string
	Branch string
	Sha    string
	Root   string
	Dirty  bool
}

type Registry struct {
	Id     string
	Region string
	Url    string
}

type Repository struct {
	Prefix string
}

type Resource struct {
	Prefix string
}

type Httproxy struct {
	ApiId *string
}

type Vpc struct {
	SecurityGroupIds []string
	SubnetIds        []string
}

type TemplateData struct {
	AccountId         string
	Region            string
	RegistryRegion    string
	RegistryAccountId string
}

type Label struct {
	Sha       string
	Role      string
	Policy    string
	Bus       string
	Resources string
}

type Labels struct {
	Schema    StringLabel
	Sha       StringLabel
	Role      EmbeddedFileLabel
	Policy    FileLabel
	Resources FileLabel
	Bus       FolderLabel
}

type Config struct {
	Function     *Function
	Functions    []Function
	Caller       Caller
	Account      Account
	Git          Git
	Registry     Registry
	Repository   Repository
	Resource     Resource
	Httproxy     Httproxy
	Vpc          Vpc
	TemplateData TemplateData
	Label        Label
	Labels       Labels
	Version      string
}

// derived information
func (c Config) ResourceName(namespace, functionName string) string {
	return c.Resource.Prefix + "-" + util.DeSlasher(namespace) + "-" + functionName
}

func (c Config) RepositoryName() string {
	return c.Repository.Prefix + "/" + c.Function.Name
}

func (c Config) RepositoryUrl() string {
	return c.Registry.Url + "/" + c.RepositoryName()
}

func (c Config) RouteKey(namespace string) string {
	verb := "ANY"
	route := "/" + c.Resource.Prefix + "/" + namespace + "/" + c.Function.Name + "/{proxy+}"
	routeKey := verb + " " + route
	return routeKey
}

// helper methods
func (c Config) Template(document string) (string, error) {
	tmpl, err := template.New("document").Parse(string(document))
	if err != nil {
		return "", err
	}

	var b strings.Builder
	if err := tmpl.Execute(&b, c.TemplateData); err != nil {
		return "", err
	}

	return b.String(), nil
}

func (c Config) AssumeRoleWithPolicy(ctx context.Context, stsc *sts.Client, policy string) (*sts.AssumeRoleOutput, error) {
	roleArn, err := util.RoleArnFromAssumeRoleArn(c.Caller.Arn)
	if err != nil {
		return nil, err
	}

	return stsc.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         aws.String(roleArn),
		RoleSessionName: aws.String(os.Getenv("USER") + "-masquerade"),
		Policy:          &policy,
	})
}

func (c Config) Json(ctx context.Context) (string, error) {
	cJson, err := json.Marshal(c)
	if err != nil {
		return "", err
	}

	return string(cJson), nil
}

func (c Config) Scaffold(templateName, functionName string) error {
	scaffoldPath := "embedded/scaffold"
	templatePath := filepath.Join(scaffoldPath, templateName)

	if _, err := embedded.ReadDir(templatePath); os.IsNotExist(err) {
		templates, err := embedded.ReadDir(scaffoldPath)
		if err != nil {
			return err
		}

		var templateNames []string
		for _, template := range templates {
			templateNames = append(templateNames, template.Name())
		}

		return fmt.Errorf("scaffold %s does not exist. valid options: %s", templateName, strings.Join(templateNames, ", "))
	}

	return fs.WalkDir(embedded, templatePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate the relative path with respect to templatePath
		relPath, err := filepath.Rel(templatePath, path)
		if err != nil {
			return err
		}
		targetFilePath := filepath.Join(functionName, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetFilePath, os.ModePerm)
		}

		content, err := fs.ReadFile(embedded, path)
		if err != nil {
			return err
		}

		tmpl, err := template.New(filepath.Base(path)).Parse(string(content))
		if err != nil {
			return err
		}

		outputFile, err := os.Create(targetFilePath)
		if err != nil {
			return err
		}
		defer outputFile.Close()

		err = tmpl.Execute(outputFile, c)
		if err != nil {
			return err
		}

		return nil
	})
}
