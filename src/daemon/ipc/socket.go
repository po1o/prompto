package ipc

import (
	"net"
)

// SocketPath returns the platform-specific socket path.
// Unix: $XDG_RUNTIME_DIR/prompto-<uid>.sock or <state>/prompto/prompto-<uid>.sock
// Windows: \\.\pipe\prompto-<username>
func SocketPath() string {
	return socketPath()
}

// Listen creates a listener on the daemon socket.
// Creates socket with proper permissions (0600 on Unix).
func Listen() (net.Listener, error) {
	return listen()
}

// CleanupSocket removes the socket file (Unix only, no-op on Windows).
func CleanupSocket() error {
	return cleanupSocket()
}

// SocketExists returns true if the daemon socket exists.
// Used to detect if the daemon is potentially running before attempting to connect.
// Implementation is platform-specific: os.Stat on Unix, pipe check on Windows.
func SocketExists() bool {
	return socketExists()
}
