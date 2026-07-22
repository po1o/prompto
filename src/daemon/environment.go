package daemon

import (
	"sync/atomic"

	"github.com/po1o/prompto/src/log"
	"github.com/po1o/prompto/src/runtime"
)

// Environment wraps runtime.Terminal to use environment variables
// from the client request instead of the daemon's own environment.
// This ensures segments see the shell's environment, not the daemon's.
//
// Boundary: only interface-level Getenv calls are intercepted. Because of
// Go embedded-method shadowing, Terminal's own internal env reads (e.g.
// PROMPTO_CURSOR_LINE, COLUMNS inside src/runtime/terminal.go) bypass this
// wrapper and still see the daemon's environment.
type Environment struct {
	*runtime.Terminal
	// envVars holds the env map of the most recent client request. It is
	// swapped by applyRenderFlags (hard cancel) and UpdateForRepaint (soft
	// cancel) while segment goroutines from a previous render generation may
	// still call Getenv concurrently, so every read and write must go
	// through the atomic pointer (Getenv / setEnvVars).
	envVars atomic.Pointer[map[string]string]
}

// NewEnvironment creates a new daemon environment.
// The envVars map contains environment variables from the client request.
func NewEnvironment(flags *runtime.Flags, envVars map[string]string) *Environment {
	term := &runtime.Terminal{}
	term.Init(flags)

	de := &Environment{
		Terminal: term,
	}
	de.setEnvVars(envVars)

	return de
}

// setEnvVars atomically replaces the request env map. A nil map means the
// request sent no environment and Getenv falls back to the daemon's own.
func (de *Environment) setEnvVars(envVars map[string]string) {
	de.envVars.Store(&envVars)
}

// Getenv returns the value of the environment variable named by the key.
// When the request supplied an env map, that map is authoritative: the
// client sends its complete environ, so a key absent from the map is unset
// in the client shell and Getenv returns the empty string. Only when the
// request sent no env map (nil) does Getenv fall back to the daemon's own
// environment.
func (de *Environment) Getenv(key string) string {
	if envVars := de.envVars.Load(); envVars != nil && *envVars != nil {
		log.Debugf("daemon env (from request): %s", key)
		return (*envVars)[key]
	}

	// No request env: fall back to the daemon's environment.
	return de.Terminal.Getenv(key)
}

// UpdateForRepaint updates the environment for a Soft-Cancel (vim toggle)
// render: only the vim-mode flag changes; the rest of the request context
// is preserved so the in-flight computations stay valid. See
// src/daemon/ARCHITECTURE.md ("The cancel model").
func (de *Environment) UpdateForRepaint(flags *runtime.Flags, envVars map[string]string) {
	if envVars != nil {
		de.setEnvVars(envVars)
	}

	if flags == nil {
		return
	}

	if de.Terminal == nil {
		return
	}

	if de.CmdFlags == nil {
		return
	}

	de.CmdFlags.VimMode = flags.VimMode
}
