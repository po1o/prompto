package upgrade

import (
	"io"
	stdhttp "net/http"
	"os"
	"strings"
	"testing"

	"github.com/po1o/prompto/src/build"
	runtimehttp "github.com/po1o/prompto/src/runtime/http"
	"github.com/stretchr/testify/assert"
)

type httpClientFunc func(req *stdhttp.Request) (*stdhttp.Response, error)

func (fn httpClientFunc) Do(req *stdhttp.Request) (*stdhttp.Response, error) {
	return fn(req)
}

func TestCanUpgrade(t *testing.T) {
	const latest = "3.1.0"

	oldHTTPClient := runtimehttp.HTTPClient
	runtimehttp.HTTPClient = httpClientFunc(func(_ *stdhttp.Request) (*stdhttp.Response, error) {
		return &stdhttp.Response{
			StatusCode: stdhttp.StatusOK,
			Body:       io.NopCloser(strings.NewReader(latest)),
		}, nil
	})
	t.Cleanup(func() {
		runtimehttp.HTTPClient = oldHTTPClient
	})

	oldIsConnected := isConnected
	isConnected = func() bool { return true }
	t.Cleanup(func() {
		isConnected = oldIsConnected
	})

	oldVersion := build.Version
	t.Cleanup(func() {
		build.Version = oldVersion
		os.Setenv("PROMPTO_INSTALLER", "")
	})

	ugc := &Config{}

	cases := []struct {
		Case           string
		CurrentVersion string
		Installer      string
		Expected       bool
	}{
		{Case: "Up to date", CurrentVersion: latest},
		{Case: "Outdated Linux", Expected: true, CurrentVersion: "3.0.0"},
		{Case: "Outdated Darwin", Expected: true, CurrentVersion: "3.0.0"},
		{Case: "Windows Store", Installer: "ws"},
	}

	for _, tc := range cases {
		build.Version = tc.CurrentVersion

		if len(tc.Installer) > 0 {
			os.Setenv("PROMPTO_INSTALLER", tc.Installer)
		}

		_, canUpgrade := ugc.Notice()
		assert.Equal(t, tc.Expected, canUpgrade, tc.Case)

		os.Setenv("PROMPTO_INSTALLER", "")
	}
}
