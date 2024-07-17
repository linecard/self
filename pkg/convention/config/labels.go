package config

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type LabelKeyContract struct {
	Schema    string
	Name      string
	Branch    string
	Sha       string
	Origin    string
	Role      string
	Policy    string
	Resources string
	Bus       string
}

type EncodedLabel struct {
	Key   string
	Value string
}

type DecodedLabel EncodedLabel

type FileLabel struct {
	Description string
	Path        string
	Key         string
	Required    bool
}

type EmbeddedFileLabel struct {
	Description string
	Path        string
	Key         string
	Required    bool
}

type FolderLabel struct {
	Description string
	Path        string
	KeyPrefix   string
	Required    bool
}

type StringLabel struct {
	Description string
	Key         string
	Content     string
	Required    bool
}

var LabelKeys = LabelKeyContract{
	Schema:    "org.linecard.self.schema",
	Name:      "org.linecard.self.name",
	Branch:    "org.linecard.self.git.branch",
	Sha:       "org.linecard.self.git.sha",
	Origin:    "org.linecard.self.git.origin",
	Role:      "org.linecard.self.role",
	Policy:    "org.linecard.self.policy",
	Resources: "org.linecard.self.resources",
	Bus:       "org.linecard.self.bus",
}

func (s StringLabel) Encode() (EncodedLabel, error) {
	if s.Required && s.Content == "" {
		return EncodedLabel{}, fmt.Errorf("label %s requirement failed", s.Key)
	}

	return EncodedLabel{
		Key:   s.Key,
		Value: base64.StdEncoding.EncodeToString([]byte(s.Content)),
	}, nil
}

func (s StringLabel) Decode(labels map[string]string) (DecodedLabel, error) {
	for k, v := range labels {
		if k == s.Key {
			decoded, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return DecodedLabel{}, err
			}

			return DecodedLabel{
				Key:   s.Key,
				Value: string(decoded),
			}, nil
		}
	}

	if s.Required {
		return DecodedLabel{}, fmt.Errorf("label %s required but not found", s.Key)
	}

	return DecodedLabel{}, nil
}

func (f EmbeddedFileLabel) Encode() (EncodedLabel, error) {
	var byteContent []byte
	var err error

	if _, err := fs.Stat(embedded, f.Path); err != nil {
		if f.Required {
			return EncodedLabel{}, err
		}

		return EncodedLabel{}, nil
	}

	byteContent, err = fs.ReadFile(embedded, f.Path)
	if err != nil {
		return EncodedLabel{}, err
	}

	if strings.Contains(f.Path, ".json") {
		compacted := new(bytes.Buffer)

		if !json.Valid(byteContent) {
			return EncodedLabel{}, fmt.Errorf("invalid JSON in %s", f.Path)
		}

		if err := json.Compact(compacted, byteContent); err != nil {
			return EncodedLabel{}, err
		}

		encoded := base64.StdEncoding.EncodeToString(compacted.Bytes())
		return EncodedLabel{
			Key:   f.Key,
			Value: encoded,
		}, nil
	}

	chomped := strings.TrimSuffix(string(byteContent), "\r\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(chomped))
	return EncodedLabel{
		Key:   f.Key,
		Value: encoded,
	}, nil
}

func (f EmbeddedFileLabel) Decode(labels map[string]string) (DecodedLabel, error) {
	for k, v := range labels {
		if k == f.Key {
			decoded, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return DecodedLabel{}, err
			}

			return DecodedLabel{
				Key:   f.Key,
				Value: string(decoded),
			}, nil
		}
	}

	if f.Required {
		return DecodedLabel{}, fmt.Errorf("label %s required but not found", f.Key)

	}

	return DecodedLabel{}, nil
}

func (f FileLabel) Encode() (EncodedLabel, error) {
	var byteContent []byte
	var err error

	if _, err := os.Stat(f.Path); err != nil {
		if f.Required {
			return EncodedLabel{}, err
		}

		return EncodedLabel{}, nil
	}

	byteContent, err = os.ReadFile(f.Path)
	if err != nil {
		return EncodedLabel{}, err
	}

	if strings.Contains(f.Path, ".json") {
		compacted := new(bytes.Buffer)

		if !json.Valid(byteContent) {
			return EncodedLabel{}, fmt.Errorf("invalid JSON in %s", f.Path)
		}

		if err := json.Compact(compacted, byteContent); err != nil {
			return EncodedLabel{}, err
		}

		encoded := base64.StdEncoding.EncodeToString(compacted.Bytes())
		return EncodedLabel{
			Key:   f.Key,
			Value: encoded,
		}, nil
	}

	chomped := strings.TrimSuffix(string(byteContent), "\r\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(chomped))
	return EncodedLabel{
		Key:   f.Key,
		Value: encoded,
	}, nil
}

func (f FileLabel) Decode(labels map[string]string) (DecodedLabel, error) {
	for k, v := range labels {
		if k == f.Key {
			decoded, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return DecodedLabel{}, err
			}

			return DecodedLabel{
				Key:   f.Key,
				Value: string(decoded),
			}, nil
		}
	}

	if f.Required {
		return DecodedLabel{}, fmt.Errorf("label %s required but not found", f.Key)
	}

	return DecodedLabel{}, nil
}

func (f FolderLabel) Encode() (encodedFiles []EncodedLabel, err error) {
	if _, err := os.Stat(f.Path); err != nil {
		if f.Required {
			return encodedFiles, err
		}

		return encodedFiles, nil
	}

	err = filepath.Walk(f.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			// extract filename without extention(s).
			filename := filepath.Base(path)
			shortfilename := strings.Split(filename, ".")[0]

			// extract dir path, drop all of path but current dir.
			dirpath := filepath.Dir(path)
			shortpath := strings.Replace(dirpath, f.Path, "", 1)

			// convert shortpath + shortfilename to dotpath.
			dotpath := strings.ReplaceAll(shortpath, "/", ".")
			dotpath = dotpath + "." + shortfilename
			dotpath = strings.TrimPrefix(dotpath, ".")
			dotpath = strings.TrimSuffix(dotpath, ".")
			label := f.KeyPrefix + "." + dotpath

			// encode file content.
			fileLabel := FileLabel{
				Description: "Individual embedded bus template",
				Path:        path,
				Key:         label,
				Required:    true,
			}

			encodedFile, err := fileLabel.Encode()
			if err != nil {
				return err
			}

			encodedFiles = append(encodedFiles, encodedFile)
		}

		return nil
	})

	if err != nil {
		return encodedFiles, err
	}

	return encodedFiles, nil
}

func (f FolderLabel) Decode(labels map[string]string) (decodedLabels []DecodedLabel, err error) {
	for k, v := range labels {
		if strings.HasPrefix(k, f.KeyPrefix) {
			decodedLabel, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return decodedLabels, err
			}

			decodedLabels = append(decodedLabels, DecodedLabel{
				Key:   k,
				Value: string(decodedLabel),
			})
		}
	}

	if f.Required && len(decodedLabels) == 0 {
		return decodedLabels, fmt.Errorf("no labels with prefix %s found, but required", f.KeyPrefix)
	}

	return decodedLabels, nil
}

func (l EncodedLabel) DecodedValue() string {
	decoded, _ := base64.StdEncoding.DecodeString(l.Value)
	return string(decoded)
}
