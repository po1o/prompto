//go:build !darwin

package runtime

import (
	"time"

	"github.com/po1o/prompto/src/log"
	"github.com/po1o/prompto/src/runtime/battery"
)

func (term *Terminal) BatteryState() (*battery.Info, error) {
	defer log.Trace(time.Now())
	info, err := battery.Get()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return info, nil
}
