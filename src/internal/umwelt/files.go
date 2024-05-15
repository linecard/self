package umwelt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Selfish(path string) *ThisFunction {
	signature := []string{"policy.json.tmpl", "Dockerfile"}

	for _, item := range signature {
		fullPath := fmt.Sprintf("%s/%s", path, item)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return nil
		}
	}

	return &ThisFunction{
		Name: filepath.Base(path),
		Path: path,
	}
}

func SelfDiscovery(gitRoot string) []ThisFunction {
	var discovered []ThisFunction

	filepath.Walk(gitRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// The final conditional in here is lame, but it's a quick way to avoid writing a .selfignore file
		if info.IsDir() && Selfish(path) != nil && !strings.Contains(path, "convention/config/static/") {
			discovered = append(discovered, ThisFunction{
				Name: filepath.Base(path),
				Path: path,
			})
		}

		return nil
	})

	return discovered
}
