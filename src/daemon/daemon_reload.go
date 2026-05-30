package daemon

import (
	"os"

	"github.com/po1o/prompto/src/config"
)

// startReloadAndWatchers wires the config-reload pipeline:
//   - capture the current root-config mtime so catch-up reloads work
//   - start a ConfigWatcher whose fsnotify events queue a coalesced reload
//   - spawn the reload worker goroutine that drains the queue
//   - start a BinaryWatcher that stops the daemon when the binary changes
//
// Called from constructors that take a configPath; safe to omit for tests.
func (daemon *Daemon) startReloadAndWatchers() {
	if daemon.configPath == "" {
		return
	}

	daemon.captureConfigModTime()

	configWatcher, err := NewConfigWatcher(daemon.requestConfigReload)
	if err == nil {
		daemon.configWatcher = configWatcher
		daemon.refreshConfigWatches()
		go daemon.configReloadWorker()
	}

	binaryPath, err := os.Executable()
	if err == nil {
		// If the binary is replaced while running, stop the daemon; the
		// next client start will launch the new one.
		watcher, watchErr := NewBinaryWatcher(binaryPath, daemon.Stop)
		if watchErr == nil {
			daemon.binaryWatcher = watcher
		}
	}
}

// ProcessPendingConfigReload drains any queued reload signals without
// blocking. Called by Server.RenderPrompt at the top of each render so the
// first prompt after a config save reflects the updated config.
func (daemon *Daemon) ProcessPendingConfigReload() {
	if daemon == nil {
		return
	}

	for {
		select {
		case <-daemon.configReloadCh:
			daemon.applyConfigReload()
		default:
			return
		}
	}
}

// ReloadIfConfigFileUpdated re-checks the root config mtime and applies a
// reload if it moved since the last applied reload. This catches saves that
// happened before the fsnotify event was delivered.
func (daemon *Daemon) ReloadIfConfigFileUpdated() {
	if daemon == nil || daemon.configPath == "" {
		return
	}

	current := configModTimeUnix(daemon.configPath)
	if current == 0 {
		return
	}

	if current <= daemon.lastConfigModUnixNano.Load() {
		return
	}

	daemon.applyConfigReload()
}

// configReloadWorker drains configReloadCh until the daemon stops. Reload
// itself is guarded by ReloadGate inside daemon.Reload, so a single worker
// keeps sequencing easy to reason about.
func (daemon *Daemon) configReloadWorker() {
	for {
		select {
		case <-daemon.done:
			return
		case <-daemon.configReloadCh:
			daemon.applyConfigReload()
		}
	}
}

func (daemon *Daemon) applyConfigReload() {
	if daemon == nil || daemon.configPath == "" {
		return
	}

	daemon.reloadMu.Lock()
	defer daemon.reloadMu.Unlock()

	// Cancel in-flight renders first so reload does not block waiting for slow segment completion.
	daemon.Reset()

	daemon.Reload(nil)

	daemon.refreshConfigWatches()
	daemon.captureConfigModTime()
}

func (daemon *Daemon) captureConfigModTime() {
	if daemon == nil || daemon.configPath == "" {
		return
	}

	daemon.lastConfigModUnixNano.Store(configModTimeUnix(daemon.configPath))
}

func configModTimeUnix(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}

	return info.ModTime().UnixNano()
}

// requestConfigReload is the ConfigWatcher callback. It coalesces bursts of
// fsnotify events: a buffered-1 channel means extra signals are dropped
// while a reload is already queued.
func (daemon *Daemon) requestConfigReload(configPath string) {
	// Ignore unrelated watched files. We only reload for this daemon's root config.
	if configPath == "" || configPath != daemon.configPath {
		return
	}

	select {
	case <-daemon.done:
		return
	default:
	}

	select {
	case daemon.configReloadCh <- struct{}{}:
	default:
	}
}

// refreshConfigWatches re-registers all resolved files (root + extends +
// symlink targets). ConfigWatcher.Watch is idempotent.
func (daemon *Daemon) refreshConfigWatches() {
	if daemon.configWatcher == nil || daemon.configPath == "" {
		return
	}

	cfg, err := config.Parse(daemon.configPath)
	if err != nil {
		return
	}

	_ = daemon.configWatcher.Watch(daemon.configPath, cfg.FilePaths)
}
