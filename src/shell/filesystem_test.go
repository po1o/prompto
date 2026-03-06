package shell

import (
	"testing"

	"github.com/po1o/prompto/src/runtime"
)

func TestCacheValueChangesWhenDaemonModeChanges(t *testing.T) {
	flagsA := &runtime.Flags{
		Shell:      ZSH,
		ConfigHash: 42,
		Daemon:     false,
		Strict:     false,
	}
	envA := &runtime.Terminal{}
	envA.Init(flagsA)

	flagsB := &runtime.Flags{
		Shell:      ZSH,
		ConfigHash: 42,
		Daemon:     true,
		Strict:     false,
	}
	envB := &runtime.Terminal{}
	envB.Init(flagsB)

	if cacheValue(envA) == cacheValue(envB) {
		t.Fatalf("expected cache value to differ when daemon mode changes")
	}
}

func TestCacheValueChangesWhenStrictModeChanges(t *testing.T) {
	flagsA := &runtime.Flags{
		Shell:      ZSH,
		ConfigHash: 42,
		Daemon:     false,
		Strict:     false,
	}
	envA := &runtime.Terminal{}
	envA.Init(flagsA)

	flagsB := &runtime.Flags{
		Shell:      ZSH,
		ConfigHash: 42,
		Daemon:     false,
		Strict:     true,
	}
	envB := &runtime.Terminal{}
	envB.Init(flagsB)

	if cacheValue(envA) == cacheValue(envB) {
		t.Fatalf("expected cache value to differ when strict mode changes")
	}
}

func TestCacheValueChangesWhenConfigPathChanges(t *testing.T) {
	flagsA := &runtime.Flags{
		Shell:      ZSH,
		ConfigPath: "/tmp/a.omp.yaml",
		ConfigHash: 42,
	}
	envA := &runtime.Terminal{}
	envA.Init(flagsA)

	flagsB := &runtime.Flags{
		Shell:      ZSH,
		ConfigPath: "/tmp/b.omp.yaml",
		ConfigHash: 42,
	}
	envB := &runtime.Terminal{}
	envB.Init(flagsB)

	if cacheValue(envA) == cacheValue(envB) {
		t.Fatalf("expected cache value to differ when init config path changes")
	}
}
