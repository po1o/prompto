package daemon

import "sync"

// ReloadGate coordinates prompt request admission with config reload operations.
// During reload, new requests are queued. Reload waits for active requests to finish.
type ReloadGate struct {
	mu        sync.Mutex
	cond      *sync.Cond
	active    int
	reloading bool
}

func NewReloadGate() *ReloadGate {
	gate := &ReloadGate{}
	gate.cond = sync.NewCond(&gate.mu)
	return gate
}

func (g *ReloadGate) StartRequest() func() {
	g.mu.Lock()
	for g.reloading {
		g.cond.Wait()
	}
	g.active++
	g.mu.Unlock()

	return func() {
		g.mu.Lock()
		if g.active > 0 {
			g.active--
		}
		g.cond.Broadcast()
		g.mu.Unlock()
	}
}

func (g *ReloadGate) BeginReload() {
	g.mu.Lock()
	for g.reloading {
		g.cond.Wait()
	}
	g.reloading = true
	for g.active > 0 {
		g.cond.Wait()
	}
	g.mu.Unlock()
}

func (g *ReloadGate) EndReload() {
	g.mu.Lock()
	g.reloading = false
	g.cond.Broadcast()
	g.mu.Unlock()
}

func (g *ReloadGate) Snapshot() (active int, reloading bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.active, g.reloading
}
