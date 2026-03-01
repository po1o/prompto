package prompt

func (e *Engine) SetUpdateCallback(callback func(string)) {
	e.stateMu.Lock()
	defer e.stateMu.Unlock()
	e.updateCallback = callback
}

func (e *Engine) notifySegmentUpdate(segmentName string) {
	e.stateMu.Lock()
	callback := e.updateCallback
	e.stateMu.Unlock()

	if callback == nil {
		return
	}

	callback(segmentName)
}
