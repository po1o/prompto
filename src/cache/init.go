package cache

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/po1o/prompto/src/log"
)

type Option func()

var (
	sessionID  string
	newSession bool
	noSession  bool
	once       sync.Once
)

var NewSession Option = func() {
	log.Debug("starting a new session")
	newSession = true
}

var Persist Option = func() {
	log.Debug("persistent cache is disabled")
}

var NoSession Option = func() {
	log.Debug("disable session cache")
	noSession = true
}

func Init(shell string, options ...Option) {
	newSession = false
	noSession = false

	for _, opt := range options {
		opt()
	}

	Device.init(DeviceStore)

	if noSession {
		return
	}

	sessionFileName := fmt.Sprintf("%s.%s.%s", shell, SessionID(), DeviceStore)
	Session.init(sessionFileName)
}

func SessionID() string {
	defer log.Trace(time.Now())

	once.Do(func() {
		if newSession {
			sessionID = uuid.NewString()
			return
		}

		sessionID = os.Getenv("PROMPTO_SESSION_ID")
		if sessionID == "" {
			sessionID = uuid.NewString()
		}
	})

	return sessionID
}

func Close() {
	Session.close()
	Device.close()
}
