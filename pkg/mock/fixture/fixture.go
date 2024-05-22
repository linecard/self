package mock

import (
	"embed"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

//go:embed fixtures/*
var fixtures embed.FS

func Copy(src, dst string) error {
	sourceFile, err := fixtures.Open("fixtures/" + src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destDir := filepath.Dir(dst)
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy contents: %w", err)
	}

	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to flush contents to disk: %w", err)
	}

	return nil
}

func Base64(src string) string {
	// Open the source file from the embedded filesystem
	sourceFile, err := fixtures.Open("fixtures/" + src)
	if err != nil {
		log.Fatalf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	// Read the file content
	content, err := io.ReadAll(sourceFile)
	if err != nil {
		log.Fatalf("failed to read source file: %v", err)
	}

	chomped := JsonChomp(string(content))

	// Encode the content to Base64
	return base64.StdEncoding.EncodeToString([]byte(chomped))
}

// this is should get refactored out, it's just a copy paste of a string manipulation that happens in labelgun
func JsonChomp(content string) string {
	for _, char := range []string{"\n", "\t"} {
		content = strings.ReplaceAll(content, char, "")
	}
	return content
}

func Read(src string) string {
	sourceFile, err := fixtures.Open("fixtures/" + src)
	if err != nil {
		log.Fatalf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	var builder strings.Builder
	if _, err := io.Copy(&builder, sourceFile); err != nil {
		log.Fatalf("failed to read fixture: %v", err)
	}

	return builder.String()
}
