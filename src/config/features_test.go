package config

import (
	"testing"

	"github.com/jandedobbeleer/oh-my-posh/src/cli/upgrade"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime/mock"
	"github.com/jandedobbeleer/oh-my-posh/src/shell"
	"github.com/stretchr/testify/assert"
)

func TestFeatures(t *testing.T) {
	cases := []struct {
		Case     string
		Shell    string
		Daemon   bool
		Async    bool
		Expected shell.Features
	}{
		{
			Case:     "Daemon enabled, supported shell",
			Shell:    shell.ZSH,
			Daemon:   true,
			Expected: shell.Daemon,
		},
		{
			Case:     "Daemon enabled, unsupported shell",
			Shell:    shell.CMD,
			Daemon:   true,
			Expected: 0,
		},
		{
			Case:     "Daemon disabled",
			Shell:    shell.ZSH,
			Daemon:   false,
			Expected: 0,
		},
		{
			Case:     "Async enabled",
			Shell:    shell.ZSH,
			Async:    true,
			Expected: shell.Async,
		},
		{
			Case:     "Async enabled, unsupported shell",
			Shell:    shell.CMD,
			Async:    true,
			Expected: 0,
		},
		{
			Case:     "Async and Daemon enabled",
			Shell:    shell.ZSH,
			Async:    true,
			Daemon:   true,
			Expected: shell.Async | shell.Daemon,
		},
	}

	for _, tc := range cases {
		env := &mock.Environment{}
		env.On("Shell").Return(tc.Shell)

		cfg := &Config{
			Async:   tc.Async,
			Upgrade: &upgrade.Config{},
		}

		got := cfg.Features(env, tc.Daemon)
		assert.Equal(t, tc.Expected, got, tc.Case)
	}
}

func TestVimFeatures(t *testing.T) {
	cases := []struct {
		Case        string
		VimEnabled  bool
		CursorShape bool
		CursorBlink bool
		ExpVimMode  bool
		ExpCursor   bool
		ExpBlink    bool
	}{
		{
			Case:       "No vim config",
			VimEnabled: false,
			ExpVimMode: false,
			ExpCursor:  false,
			ExpBlink:   false,
		},
		{
			Case:       "Vim enabled only",
			VimEnabled: true,
			ExpVimMode: true,
			ExpCursor:  false,
			ExpBlink:   false,
		},
		{
			Case:        "Cursor shape implies vim mode",
			VimEnabled:  false,
			CursorShape: true,
			ExpVimMode:  true,
			ExpCursor:   true,
			ExpBlink:    false,
		},
		{
			Case:        "Cursor blink implies vim mode",
			VimEnabled:  false,
			CursorBlink: true,
			ExpVimMode:  true,
			ExpCursor:   true,
			ExpBlink:    true,
		},
		{
			Case:        "Both enabled",
			VimEnabled:  true,
			CursorShape: true,
			ExpVimMode:  true,
			ExpCursor:   true,
			ExpBlink:    false,
		},
		{
			Case:        "Shape and blink enabled",
			VimEnabled:  true,
			CursorShape: true,
			CursorBlink: true,
			ExpVimMode:  true,
			ExpCursor:   true,
			ExpBlink:    true,
		},
	}

	for _, tc := range cases {
		env := &mock.Environment{}
		env.On("Shell").Return(shell.ZSH)

		var vim *VimConfig
		if tc.VimEnabled || tc.CursorShape || tc.CursorBlink {
			vim = &VimConfig{
				Enabled:     tc.VimEnabled,
				CursorShape: tc.CursorShape,
				CursorBlink: tc.CursorBlink,
			}
		}

		cfg := &Config{
			Upgrade: &upgrade.Config{},
			Vim:     vim,
		}

		got := cfg.Features(env, false)

		hasVimMode := got&shell.VimMode != 0
		hasCursor := got&shell.VimCursorShape != 0
		hasBlink := got&shell.VimCursorBlink != 0

		assert.Equal(t, tc.ExpVimMode, hasVimMode, tc.Case+" - VimMode")
		assert.Equal(t, tc.ExpCursor, hasCursor, tc.Case+" - CursorShape")
		assert.Equal(t, tc.ExpBlink, hasBlink, tc.Case+" - CursorBlink")
	}
}
