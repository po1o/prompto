package daemon

import (
	"sync"
	"time"

	"github.com/jandedobbeleer/oh-my-posh/src/config"
)

// CachedConfig holds a parsed config with metadata.
type CachedConfig struct {
	LoadedAt  time.Time
	Config    *config.Config
	FilePaths []string
	Hash      uint64
}

// ConfigCache manages cached configs by path.
// Thread-safe for concurrent access.
type ConfigCache struct {
	configs map[string]*CachedConfig
	mu      sync.RWMutex
}

// NewConfigCache creates a new empty config cache.
func NewConfigCache() *ConfigCache {
	return &ConfigCache{
		configs: make(map[string]*CachedConfig),
	}
}

// Get retrieves a cached config by path.
// Returns nil and false if not found.
func (c *ConfigCache) Get(configPath string) (*CachedConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.configs[configPath]
	if !ok {
		return nil, false
	}

	return cached, true
}

// Set stores a config in the cache.
func (c *ConfigCache) Set(configPath string, cfg *config.Config, filePaths []string) *CachedConfig {
	cached := &CachedConfig{
		Config:    cfg,
		Hash:      cfg.Hash(),
		LoadedAt:  time.Now(),
		FilePaths: filePaths,
	}

	c.mu.Lock()
	c.configs[configPath] = cached
	c.mu.Unlock()

	return cached
}

// Invalidate removes a config from the cache.
// Called when the config file changes.
func (c *ConfigCache) Invalidate(configPath string) {
	c.mu.Lock()
	delete(c.configs, configPath)
	c.mu.Unlock()
}

// InvalidateAll removes all configs from the cache.
func (c *ConfigCache) InvalidateAll() {
	c.mu.Lock()
	c.configs = make(map[string]*CachedConfig)
	c.mu.Unlock()
}

// Count returns the number of cached configs.
func (c *ConfigCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.configs)
}
