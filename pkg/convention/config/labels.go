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

func (s StringLabel) Encode() (map[string]string, error) {
	if s.Required && s.Content == "" {
		return map[string]string{}, fmt.Errorf("label %s requirement failed", s.Key)
	}

	return map[string]string{
		s.Key: base64.StdEncoding.EncodeToString([]byte(s.Content)),
	}, nil
}

func (s StringLabel) Decode(labels map[string]string) (map[string]string, error) {
	for k, v := range labels {
		if k == s.Key {
			decoded, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return map[string]string{}, err
			}

			return map[string]string{
				s.Key: string(decoded),
			}, nil
		}
	}

	if s.Required {
		return map[string]string{}, fmt.Errorf("label %s not found", s.Key)
	}

	return map[string]string{}, nil
}

func (f EmbeddedFileLabel) Encode() (map[string]string, error) {
	var byteContent []byte
	var err error

	if _, err := fs.Stat(embedded, f.Path); err != nil {
		if f.Required {
			return map[string]string{}, err
		}

		return map[string]string{}, nil
	}

	byteContent, err = fs.ReadFile(embedded, f.Path)
	if err != nil {
		return map[string]string{}, err
	}

	if strings.Contains(f.Path, ".json") {
		compacted := new(bytes.Buffer)

		if !json.Valid(byteContent) {
			return map[string]string{}, fmt.Errorf("invalid JSON in %s", f.Path)
		}

		if err := json.Compact(compacted, byteContent); err != nil {
			return map[string]string{}, err
		}

		encoded := base64.StdEncoding.EncodeToString(compacted.Bytes())
		return map[string]string{
			f.Key: encoded,
		}, nil
	}

	chomped := strings.TrimSuffix(string(byteContent), "\r\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(chomped))
	return map[string]string{
		f.Key: encoded,
	}, nil
}

func (f EmbeddedFileLabel) Decode(labels map[string]string) (map[string]string, error) {
	for k, v := range labels {
		if k == f.Key {
			decoded, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return map[string]string{}, err
			}

			return map[string]string{
				f.Key: string(decoded),
			}, nil
		}
	}

	if f.Required {
		return map[string]string{}, fmt.Errorf("label %s not found", f.Key)
	}

	return map[string]string{}, nil
}

func (f FileLabel) Encode() (map[string]string, error) {
	var byteContent []byte
	var err error

	if _, err := os.Stat(f.Path); err != nil {
		if f.Required {
			return map[string]string{}, err
		}

		return map[string]string{}, nil
	}

	byteContent, err = os.ReadFile(f.Path)
	if err != nil {
		return map[string]string{}, err
	}

	if strings.Contains(f.Path, ".json") {
		compacted := new(bytes.Buffer)

		if !json.Valid(byteContent) {
			return map[string]string{}, fmt.Errorf("invalid JSON in %s", f.Path)
		}

		if err := json.Compact(compacted, byteContent); err != nil {
			return map[string]string{}, err
		}

		encoded := base64.StdEncoding.EncodeToString(compacted.Bytes())
		return map[string]string{
			f.Key: encoded,
		}, nil
	}

	chomped := strings.TrimSuffix(string(byteContent), "\r\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(chomped))
	return map[string]string{
		f.Key: encoded,
	}, nil
}

func (f FileLabel) Decode(labels map[string]string) (map[string]string, error) {
	for k, v := range labels {
		if k == f.Key {
			decoded, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return map[string]string{}, err
			}

			return map[string]string{
				f.Key: string(decoded),
			}, nil
		}
	}

	if f.Required {
		return map[string]string{}, fmt.Errorf("label %s not found", f.Key)
	}

	return map[string]string{}, nil
}

func (f FolderLabel) Encode() (map[string]string, error) {
	encoded := make(map[string]string)

	if _, err := os.Stat(f.Path); err != nil {
		if f.Required {
			return map[string]string{}, err
		}

		return map[string]string{}, nil
	}

	err := filepath.Walk(f.Path, func(path string, info os.FileInfo, err error) error {
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

			content, err := fileLabel.Encode()
			if err != nil {
				return err
			}

			for k, v := range content {
				encoded[k] = v
			}
		}

		return nil
	})

	if err != nil {
		return map[string]string{}, err
	}

	return encoded, nil
}

func (f FolderLabel) Decode(labels map[string]string) (map[string]string, error) {
	decoded := make(map[string]string)

	for k, v := range labels {
		if strings.HasPrefix(k, f.KeyPrefix) {
			decodedValue, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return map[string]string{}, err
			}

			decoded[k] = string(decodedValue)
		}
	}

	if f.Required && len(decoded) == 0 {
		return map[string]string{}, fmt.Errorf("label %s not found", f.KeyPrefix)
	}

	return decoded, nil
}

func (l Labels) Encode() (map[string]string, error) {
	schema, err := l.Schema.Encode()
	if err != nil {
		return map[string]string{}, err
	}

	sha, err := l.Sha.Encode()
	if err != nil {
		return map[string]string{}, err
	}

	role, err := l.Role.Encode()
	if err != nil {
		return map[string]string{}, err
	}

	policy, err := l.Policy.Encode()
	if err != nil {
		return map[string]string{}, err
	}

	resources, err := l.Resources.Encode()
	if err != nil {
		return map[string]string{}, err
	}

	buses, err := l.Bus.Encode()
	if err != nil {
		return map[string]string{}, err
	}

	encoded := make(map[string]string)

	for k, v := range schema {
		encoded[k] = v
	}

	for k, v := range sha {
		encoded[k] = v
	}

	for k, v := range role {
		encoded[k] = v
	}

	for k, v := range policy {
		encoded[k] = v
	}

	for k, v := range resources {
		encoded[k] = v
	}

	for k, v := range buses {
		encoded[k] = v
	}

	return encoded, nil
}

func (l Labels) Decode(labels map[string]string) (map[string]string, error) {
	decoded := make(map[string]string)

	schema, err := l.Schema.Decode(labels)
	if err != nil {
		return map[string]string{}, err
	}

	sha, err := l.Sha.Decode(labels)
	if err != nil {
		return map[string]string{}, err
	}

	role, err := l.Role.Decode(labels)
	if err != nil {
		return map[string]string{}, err
	}

	policy, err := l.Policy.Decode(labels)
	if err != nil {
		return map[string]string{}, err
	}

	resources, err := l.Resources.Decode(labels)
	if err != nil {
		return map[string]string{}, err
	}

	bus, err := l.Bus.Decode(labels)
	if err != nil {
		return map[string]string{}, err
	}

	for k, v := range schema {
		decoded[k] = v
	}

	for k, v := range sha {
		decoded[k] = v
	}

	for k, v := range role {
		decoded[k] = v
	}

	for k, v := range policy {
		decoded[k] = v
	}

	for k, v := range resources {
		decoded[k] = v
	}

	for k, v := range bus {
		decoded[k] = v
	}

	return decoded, nil
}
