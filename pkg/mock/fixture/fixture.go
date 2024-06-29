package mock

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
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
		log.Fatal().Err(err).Msgf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	// Read the file content
	content, err := io.ReadAll(sourceFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to read source file: %v", err)
	}

	compacted, err := JsonCompact(content)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to compact JSON: %v", err)
	}

	// Encode the content to Base64
	return base64.StdEncoding.EncodeToString(compacted)
}

func JsonCompact(byteContent []byte) ([]byte, error) {
	compacted := new(bytes.Buffer)

	if !json.Valid(byteContent) {
		return []byte{}, fmt.Errorf("invalid JSON in fixture")
	}

	if err := json.Compact(compacted, byteContent); err != nil {
		return []byte{}, err
	}

	return compacted.Bytes(), nil
}

func Read(src string) string {
	sourceFile, err := fixtures.Open("fixtures/" + src)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open source file")
	}
	defer sourceFile.Close()

	var builder strings.Builder
	if _, err := io.Copy(&builder, sourceFile); err != nil {
		log.Fatal().Err(err).Msg("failed to read source file")
	}

	return builder.String()
}
