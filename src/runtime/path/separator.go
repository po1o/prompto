package path

import (
	"time"

	"github.com/po1o/prompto/src/log"
)

func Separator() string {
	defer log.Trace(time.Now())

	return "/"
}

func IsSeparator(c uint8) bool {
	return c == '/'
}
