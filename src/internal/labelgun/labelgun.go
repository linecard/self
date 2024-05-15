package labelgun

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func EncodeString(content string, isJson bool) (string, error) {
	if isJson {
		if !json.Valid([]byte(content)) {
			return "", fmt.Errorf("invalid JSON")
		}
		content = JsonChomp(content)
	}

	return base64.StdEncoding.EncodeToString([]byte(content)), nil
}

func EncodeFile(path string) (string, error) {
	isJson := (strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".json.tmpl"))

	if _, err := os.Stat(path); err != nil {
		return "", err
	}

	byteContent, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	content := string(byteContent)

	if isJson {
		if !json.Valid(byteContent) {
			return "", fmt.Errorf("invalid JSON in %s", path)
		}
		content = JsonChomp(content)
	}

	contentBase64 := base64.StdEncoding.EncodeToString([]byte(content))
	return contentBase64, nil
}

func EncodePath(labelPrefix, dirPath string) (map[string]string, error) {
	labels := make(map[string]string)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			// extract filename without extention(s).
			filename := filepath.Base(path)
			shortfilename := strings.Split(filename, ".")[0]

			// extract dir path, drop all of path but current dir.
			dirpath := filepath.Dir(path)
			shortpath := strings.Replace(dirpath, dirPath, "", 1)

			// convert shortpath + shortfilename to dotpath.
			dotpath := strings.ReplaceAll(shortpath, "/", ".")
			dotpath = dotpath + "." + shortfilename
			dotpath = strings.TrimPrefix(dotpath, ".")
			dotpath = strings.TrimSuffix(dotpath, ".")

			label := labelPrefix + "." + dotpath

			// encode file content.
			content, err := EncodeFile(path)
			if err != nil {
				return err
			}

			labels[label] = content
		}

		return nil
	})

	if err != nil {
		return make(map[string]string), err
	}

	return labels, nil
}

func DecodeLabel(label string, labels map[string]string) (string, error) {
	value, ok := labels[label]
	if !ok {
		return "", fmt.Errorf("label not found %s", label)
	}

	contentBytes, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}

	return string(contentBytes), nil
}

func DecodeLabels(prefix string, labels map[string]string) (map[string]string, error) {
	decoded := make(map[string]string)
	for label, rawValue := range labels {
		if strings.HasPrefix(label, prefix) {
			// Until all labels are migrated to being always base64, fail down to plaintext
			decodedValue, err := DecodeLabel(label, labels)
			if err != nil {
				decoded[label] = rawValue
			} else {
				decoded[label] = decodedValue
			}
		}
	}

	return decoded, nil
}

func HasLabel(label string, labels map[string]string) bool {
	_, ok := labels[label]
	return ok
}

// Replace the usage of this with util.Chomp. This is an unecessary optimization.
func JsonChomp(content string) string {
	for _, char := range []string{"\n", "\t"} {
		content = strings.ReplaceAll(content, char, "")
	}
	return content
}
