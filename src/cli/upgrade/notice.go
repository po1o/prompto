package upgrade

import (
	"fmt"
	"os"

	"github.com/po1o/prompto/src/build"
	"github.com/po1o/prompto/src/log"
	"github.com/po1o/prompto/src/runtime/http"
)

var isConnected = http.IsConnected

const (
	CACHEKEY = "upgrade_check"

	upgradeNotice = `
A new release of Prompto is available: v%s → v%s
To upgrade, run: 'prompto upgrade%s'

To enable automated upgrades, run: 'prompto enable upgrade'.
`
)

// Returns the upgrade notice if a new version is available
// that should be displayed to the user.
//
// The upgrade check is only performed every other week.
func (cfg *Config) Notice() (string, bool) {
	// never validate when we install using the Windows Store
	if os.Getenv("PROMPTO_INSTALLER") == "ws" {
		log.Debug("skipping upgrade check because we are using the Windows Store")
		return "", false
	}

	if !isConnected() {
		return "", false
	}

	latest, err := cfg.FetchLatest()
	if err != nil {
		return "", false
	}

	if latest == build.Version {
		return "", false
	}

	var forceUpdate string
	if IsMajorUpgrade(build.Version, latest) {
		forceUpdate = " --force"
	}

	return fmt.Sprintf(upgradeNotice, build.Version, latest, forceUpdate), true
}
