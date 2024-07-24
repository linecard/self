package manifest

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type StringLabel struct {
	Description string
	Encoded     string
	Decoded     string
	Key         string
	Required    bool
}

type FileLabel struct {
	Description string
	Decoded     string
	Encoded     string
	Key         string
	Required    bool
}

type EmbeddedFileLabel struct {
	Description string
	Decoded     string
	Encoded     string
	Key         string
	Required    bool
}

type FolderLabel struct {
	Description string
	Content     []FileLabel
	KeyPrefix   string
	Required    bool
}

func templateString(content string, data any) (string, error) {
	tmpl, err := template.New("label").Parse(content)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (s *StringLabel) Encode(content string) error {
	if s.Required && content == "" {
		return fmt.Errorf("label %s requirement failed", s.Key)
	}
	s.Decoded = content
	s.Encoded = base64.StdEncoding.EncodeToString([]byte(content))
	return nil
}

func (s *StringLabel) Decode(labels map[string]string) error {
	for k, v := range labels {
		if k == s.Key {
			decoded, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return err
			}
			s.Encoded = v
			s.Decoded = string(decoded)
			return nil
		}
	}

	if s.Required {
		return fmt.Errorf("label %s required but not found", s.Key)
	}

	return nil
}

func (s *StringLabel) Template(data any) (err error) {
	if s.Decoded, err = templateString(s.Decoded, data); err != nil {
		return err
	}
	return nil
}

func (f *EmbeddedFileLabel) Encode(path string) error {
	var byteContent []byte
	var err error

	if _, err := fs.Stat(embedded, path); err != nil {
		if f.Required {
			return err
		}

		return nil
	}

	byteContent, err = fs.ReadFile(embedded, path)
	if err != nil {
		return err
	}

	if strings.Contains(path, ".json") {
		if !json.Valid(byteContent) {
			return fmt.Errorf("invalid JSON in %s", path)
		}

		compacted := Compact(byteContent)

		f.Decoded = string(compacted)
		f.Encoded = base64.StdEncoding.EncodeToString(compacted)
		return nil
	}

	chomped := strings.TrimSuffix(string(byteContent), "\r\n")
	chomped = strings.TrimPrefix(chomped, "\r\n")
	f.Decoded = chomped
	f.Encoded = base64.StdEncoding.EncodeToString([]byte(chomped))
	return nil
}

func (f *EmbeddedFileLabel) Decode(labels map[string]string) error {
	for k, v := range labels {
		if k == f.Key {
			decoded, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return err
			}

			f.Encoded = v
			f.Decoded = string(decoded)
			return nil
		}
	}

	if f.Required {
		return fmt.Errorf("label %s required but not found", f.Key)
	}

	return nil
}

func (f *EmbeddedFileLabel) Template(data any) (err error) {
	if f.Decoded, err = templateString(f.Decoded, data); err != nil {
		return err
	}
	return nil
}

func (f *FileLabel) Encode(path string) error {
	var byteContent []byte
	var err error

	if _, err := os.Stat(path); err != nil {
		if f.Required {
			return fmt.Errorf("file %s required but not found", path)
		}

		return nil
	}

	byteContent, err = os.ReadFile(path)
	if err != nil {
		return err
	}

	if strings.Contains(path, ".json") {
		if !json.Valid(byteContent) {
			return fmt.Errorf("invalid JSON in %s", path)
		}

		compacted := Compact(byteContent)

		f.Decoded = string(compacted)
		f.Encoded = base64.StdEncoding.EncodeToString(compacted)
		return nil
	}

	chomped := strings.TrimSuffix(string(byteContent), "\r\n")
	chomped = strings.TrimPrefix(chomped, "\r\n")
	f.Decoded = chomped
	f.Encoded = base64.StdEncoding.EncodeToString([]byte(chomped))
	return nil
}

func (f *FileLabel) Decode(labels map[string]string) error {
	for k, v := range labels {
		if k == f.Key {
			decoded, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return err
			}

			f.Encoded = v
			f.Decoded = string(decoded)
			return nil
		}
	}

	if f.Required {
		return fmt.Errorf("label %s required but not found", f.Key)
	}

	return nil
}

func (f *FileLabel) Template(data any) (err error) {
	if f.Decoded, err = templateString(f.Decoded, data); err != nil {
		return err
	}
	return nil
}

func (f *FolderLabel) Encode(parentPath string) error {
	if _, err := os.Stat(parentPath); err != nil {
		if f.Required {
			return err
		}

		return nil
	}

	err := filepath.Walk(parentPath, func(childPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			// extract filename without extention(s).
			filename := filepath.Base(childPath)
			shortfilename := strings.Split(filename, ".")[0]

			// extract dir path, drop all of path but current dir.
			dirpath := filepath.Dir(childPath)
			shortpath := strings.Replace(dirpath, parentPath, "", 1)

			// convert shortpath + shortfilename to dotpath.
			dotpath := strings.ReplaceAll(shortpath, "/", ".")
			dotpath = dotpath + "." + shortfilename
			dotpath = strings.TrimPrefix(dotpath, ".")
			dotpath = strings.TrimSuffix(dotpath, ".")
			label := f.KeyPrefix + "." + dotpath

			// encode file content.
			encodedFile := FileLabel{
				Description: "Individual embedded bus template",
				Key:         label,
				Required:    true,
			}

			if err := encodedFile.Encode(childPath); err != nil {
				return err
			}

			f.Content = append(f.Content, encodedFile)
		}

		return nil
	})

	if err != nil {
		return err
	}

	if f.Required && len(f.Content) == 0 {
		return fmt.Errorf("no files found in %s", parentPath)
	}

	return nil
}

func (f FolderLabel) Decode(labels map[string]string) error {
	for k, v := range labels {
		if strings.HasPrefix(k, f.KeyPrefix) {
			decodedLabel, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				return err
			}

			f.Content = append(f.Content, FileLabel{
				Description: "Individual embedded bus template",
				Key:         k,
				Encoded:     v,
				Decoded:     string(decodedLabel),
			})
		}
	}

	if f.Required && len(f.Content) == 0 {
		return fmt.Errorf("no labels with prefix %s found, but required", f.KeyPrefix)
	}

	return nil
}

func (fldr *FolderLabel) Template(data any) (err error) {
	for _, f := range fldr.Content {
		if f.Decoded, err = templateString(f.Decoded, data); err != nil {
			return err
		}
	}
	return nil
}

// Lifted from https://github.com/tidwall/pretty/blob/master/pretty.go
func Compact(json []byte) []byte {
	buf := make([]byte, 0, len(json))
	return compact(buf, json)
}

func compact(dst, src []byte) []byte {
	dst = dst[:0]
	for i := 0; i < len(src); i++ {
		if src[i] > ' ' {
			dst = append(dst, src[i])
			if src[i] == '"' {
				for i = i + 1; i < len(src); i++ {
					dst = append(dst, src[i])
					if src[i] == '"' {
						j := i - 1
						for ; ; j-- {
							if src[j] != '\\' {
								break
							}
						}
						if (j-i)%2 != 0 {
							break
						}
					}
				}
			}
		}
	}
	return dst
}
