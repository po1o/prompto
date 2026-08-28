package cache

import (
	"github.com/po1o/prompto/src/log"
)

type Option func()

var noSession bool

var Persist Option = func() {
	log.Debug("persistent cache is disabled")
}

var NoSession Option = func() {
	log.Debug("disable session cache")
	noSession = true
}

func Init(options ...Option) {
	noSession = false

	for _, opt := range options {
		opt()
	}

	Device.init()

	if noSession {
		return
	}

	Session.init()
}

func Close() {
	Session.close()
	Device.close()
}
