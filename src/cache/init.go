package cache

// Init initializes the process-wide cache stores.
func Init() {
	Device.init()
	Session.init()
}
