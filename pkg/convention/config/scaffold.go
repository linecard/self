package config

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

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
