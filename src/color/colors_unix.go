//go:build !windows && !darwin

package color

import "github.com/po1o/prompto/src/runtime"

func GetAccentColor(_ runtime.Environment) (*RGB, error) {
	return nil, &runtime.NotImplemented{}
}
