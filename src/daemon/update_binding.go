package daemon

type updateCallbackSetter interface {
	SetUpdateCallback(func(string))
}

func BindSegmentUpdates(sessionID string, engine updateCallbackSetter, sessions *PromptSessionStore) {
	if engine == nil || sessions == nil {
		return
	}

	hub := sessions.Hub(sessionID)
	engine.SetUpdateCallback(func(segmentName string) {
		hub.Publish(segmentName)
	})
}

func ClearSegmentUpdates(engine updateCallbackSetter) {
	if engine == nil {
		return
	}

	engine.SetUpdateCallback(nil)
}
