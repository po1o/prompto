package shell

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"

	"github.com/po1o/prompto/src/build"
	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/log"
	"github.com/po1o/prompto/src/runtime"
)

var scriptPathCache string

func hasScript(env runtime.Environment) (string, bool) {
	if env.Flags().Debug || env.Flags().Eval {
		log.Debug("in debug or eval mode, no script path will be used")
		return "", false
	}

	path := scriptPath(env)

	_, err := os.Stat(path)
	if err != nil {
		log.Debug("script path does not exist")
		return "", false
	}

	// check if we have the same context
	if val, _ := cache.Get[string](cache.Device, cacheKey(env.Flags().Shell)); val != cacheValue(env) {
		log.Debug("script context has changed")
		return "", false
	}

	log.Debug("script context is unchanged")
	return path, true
}

func writeScript(env runtime.Environment, script string) (string, error) {
	path := scriptPath(env)

	err := os.WriteFile(path, []byte(script), 0o644)
	if err != nil {
		log.Error(err)
		return "", err
	}

	log.Debug("init script written successfully")
	cache.Set(cache.Device, cacheKey(env.Flags().Shell), cacheValue(env), cache.INFINITE)

	return path, nil
}

func cacheKey(sh string) string {
	return fmt.Sprintf("INITVERSION%s", strings.ToUpper(sh))
}

func cacheValue(env runtime.Environment) string {
	executable, err := getExecutablePath(env)
	if err != nil {
		executable = "unknown"
	}

	initSig := initCommandSignature(env.Flags())
	scriptFingerprint := shellScriptFingerprint(env.Flags().Shell)

	return fmt.Sprintf(
		"%d%s|exe=%s|init=%s|script=%d",
		env.Flags().ConfigHash,
		build.Version,
		executable,
		initSig,
		scriptFingerprint,
	)
}

func initCommandSignature(flags *runtime.Flags) string {
	return fmt.Sprintf(
		"shell=%s|config=%s|strict=%t|daemon=%t",
		flags.Shell,
		flags.ConfigPath,
		flags.Strict,
		flags.Daemon,
	)
}

func shellScriptFingerprint(sh string) uint64 {
	template := shellTemplate(sh)
	h := fnv.New64a()
	_, _ = h.Write([]byte(template))
	return h.Sum64()
}

func shellTemplate(sh string) string {
	switch sh {
	case PWSH:
		return pwshInit
	case ZSH:
		return zshInit
	case BASH:
		return bashInit
	case FISH:
		return fishInit
	default:
		return ""
	}
}

func InitScriptName(flags *runtime.Flags) string {
	sh := flags.Shell
	switch flags.Shell {
	case PWSH:
		sh = "ps1"
	case BASH:
		sh = "sh"
	}

	// to avoid a single init scripts for different configs
	// we hash the config path as part of the script name
	// that way we have a single init script per config
	// avoiding conflicts
	h := fnv.New64a()
	h.Write([]byte(flags.ConfigPath))
	hash := h.Sum64()

	return fmt.Sprintf("init.%d.%s", hash, sh)
}

func scriptPath(env runtime.Environment) string {
	if len(scriptPathCache) != 0 {
		return scriptPathCache
	}

	scriptPathCache = filepath.Join(cache.Path(), InitScriptName(env.Flags()))
	log.Debug("init script path:", scriptPathCache)
	return scriptPathCache
}
