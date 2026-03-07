package segments

import (
	"io"
	stdhttp "net/http"
	"strings"
	"testing"

	"github.com/po1o/prompto/src/build"
	"github.com/po1o/prompto/src/cache"
	runtimehttp "github.com/po1o/prompto/src/runtime/http"
	"github.com/po1o/prompto/src/runtime/mock"
	"github.com/po1o/prompto/src/segments/options"

	"github.com/alecthomas/assert"
)

type upgradeHTTPClientFunc func(req *stdhttp.Request) (*stdhttp.Response, error)

func (fn upgradeHTTPClientFunc) Do(req *stdhttp.Request) (*stdhttp.Response, error) {
	return fn(req)
}

func TestUpgrade(t *testing.T) {
	const latest = "1.0.3"

	oldHTTPClient := runtimehttp.HTTPClient
	runtimehttp.HTTPClient = upgradeHTTPClientFunc(func(_ *stdhttp.Request) (*stdhttp.Response, error) {
		return &stdhttp.Response{
			StatusCode: stdhttp.StatusOK,
			Body:       io.NopCloser(strings.NewReader(latest)),
		}, nil
	})
	t.Cleanup(func() {
		runtimehttp.HTTPClient = oldHTTPClient
		cache.DeleteAll(cache.Device)
	})

	oldVersion := build.Version
	t.Cleanup(func() {
		build.Version = oldVersion
	})

	cases := []struct {
		Case            string
		CurrentVersion  string
		LatestVersion   string
		CachedVersion   string
		ExpectedEnabled bool
		HasCache        bool
	}{
		{
			Case:            "Should upgrade",
			CurrentVersion:  "1.0.0",
			LatestVersion:   "1.0.1",
			ExpectedEnabled: true,
		},
		{
			Case:           "On latest",
			CurrentVersion: latest,
		},
		{
			Case:            "On previous, from cache",
			HasCache:        true,
			CurrentVersion:  "1.0.2",
			LatestVersion:   latest,
			CachedVersion:   "1.0.2",
			ExpectedEnabled: true,
		},
		{
			Case:           "On latest, version changed",
			HasCache:       true,
			CurrentVersion: latest,
			LatestVersion:  latest,
			CachedVersion:  "1.0.1",
		},
		{
			Case:            "On previous, version changed",
			HasCache:        true,
			CurrentVersion:  "1.0.2",
			LatestVersion:   latest,
			CachedVersion:   "1.0.1",
			ExpectedEnabled: true,
		},
	}

	for _, tc := range cases {
		env := new(mock.Environment)

		if tc.CachedVersion == "" {
			tc.CachedVersion = tc.CurrentVersion
		}

		if tc.HasCache {
			data := &UpgradeCache{
				Latest:  tc.LatestVersion,
				Current: tc.CachedVersion,
			}
			cache.Set(cache.Device, UPGRADECACHEKEY, data, cache.INFINITE)
		}

		build.Version = tc.CurrentVersion

		ug := &Upgrade{}
		ug.Init(options.Map{}, env)

		enabled := ug.Enabled()

		assert.Equal(t, tc.ExpectedEnabled, enabled, tc.Case)

		cache.DeleteAll(cache.Device)
	}
}
