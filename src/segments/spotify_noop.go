//go:build !linux && !darwin

package segments

func (s *Spotify) Enabled() bool {
	return false
}
