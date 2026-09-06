package cache

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/po1o/prompto/src/log"
)

// ClearInit removes cached shell init scripts from the cache directory.
func ClearInit() error {
	defer log.Trace(time.Now())

	cacheDir := Path()
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasPrefix(entry.Name(), "init.") {
			path := filepath.Join(cacheDir, entry.Name())
			if err := os.Remove(path); err != nil {
				log.Error(err)
				continue
			}
			log.Debugf("removed cached init script: %s", path)
		}
	}

	return nil
}
