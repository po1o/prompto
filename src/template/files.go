package template

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/po1o/prompto/src/log"
)

func glob(pattern string) (bool, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return false, err
	}
	return len(matches) > 0, nil
}

func readFile(path string) string {
	content, _ := os.ReadFile(path)
	return string(content)
}

func stat(path string) string {
	fullPath, err := exec.LookPath(path)
	if err != nil {
		log.Error(err)
		return ""
	}

	return fullPath
}
