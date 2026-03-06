package upgrade

import (
	"context"
	"encoding/gob"
	"fmt"
	"io"
	httplib "net/http"
	"strings"

	"github.com/po1o/prompto/src/cache"
	"github.com/po1o/prompto/src/cli/progress"
	"github.com/po1o/prompto/src/log"
	"github.com/po1o/prompto/src/runtime/http"
)

func init() {
	gob.Register(&Config{})
	gob.Register((*Source)(nil))
}

type Config struct {
	Source        Source         `json:"source" toml:"source" yaml:"source"`
	Interval      cache.Duration `json:"interval" toml:"interval" yaml:"interval"`
	Latest        string         `json:"-" toml:"-" yaml:"-"`
	Auto          bool           `json:"auto" toml:"auto" yaml:"auto"`
	DisplayNotice bool           `json:"notice" toml:"notice" yaml:"notice"`
	Force         bool           `json:"-" toml:"-" yaml:"-"`
}

type Source string

const (
	GitHub Source = "github"
	CDN    Source = "cdn"
)

func (s Source) String() string {
	switch s {
	case GitHub:
		return "github.com"
	case CDN:
		return "cdn.prompto.dev"
	default:
		return "Unknown"
	}
}

func (cfg *Config) FetchLatest() (string, error) {
	cfg.Latest = "latest"
	v, err := cfg.DownloadAsset("version.txt")
	if err != nil {
		log.Debugf("failed to get latest version for source: %s", cfg.Source)
		return "", err
	}

	version := strings.TrimSpace(string(v))
	cfg.Latest = version

	version = strings.TrimPrefix(version, "v")
	log.Debugf("latest version: %s", version)

	return version, err
}

func (cfg *Config) DownloadAsset(asset string) ([]byte, error) {
	if cfg.Source == "" {
		log.Debug("no source specified, defaulting to github")
		cfg.Source = GitHub
	}

	switch cfg.Source {
	case GitHub:
		var url string

		switch cfg.Latest {
		case "latest":
			url = fmt.Sprintf("https://github.com/po1o/prompto/releases/latest/download/%s", asset)
		default:
			url = fmt.Sprintf("https://github.com/po1o/prompto/releases/download/%s/%s", cfg.Latest, asset)
		}

		return cfg.Download(url)
	case CDN:
		fallthrough
	default:
		url := fmt.Sprintf("https://cdn.prompto.dev/releases/%s/%s", cfg.Latest, asset)
		return cfg.Download(url)
	}
}

func (cfg *Config) Download(url string) ([]byte, error) {
	req, err := httplib.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		log.Debugf("failed to create request for url: %s", url)
		return nil, err
	}

	req.Header.Add("User-Agent", "prompto")
	req.Header.Add("Cache-Control", "max-age=0")

	resp, err := http.HTTPClient.Do(req)
	if err != nil {
		log.Debugf("failed to execute HTTP request: %s", url)
		return nil, err
	}

	if resp.StatusCode != httplib.StatusOK {
		return nil, fmt.Errorf("failed to download asset: %s", url)
	}

	defer resp.Body.Close()

	reader := progress.NewReader(resp.Body, resp.ContentLength, program)

	data, err := io.ReadAll(reader)
	if err != nil {
		log.Debugf("failed to read response body: %s", url)
		return nil, err
	}

	return data, nil
}
