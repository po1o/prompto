package daemon

// CancelKind classifies why an in-flight render is being interrupted. It is
// the single signal the render path uses to decide whether to abort or
// preserve in-flight segment computations. See ARCHITECTURE.md, "The cancel
// model".
type CancelKind int

const (
	// CancelHard means the on-disk state may have changed — a new command
	// ran, the working directory changed. In-flight computations must be
	// aborted and must not write their results to the cache.
	CancelHard CancelKind = iota

	// CancelSoft means only the view changed, not the on-disk state — a vim
	// mode toggle / repaint. In-flight computations are preserved and their
	// results reused.
	CancelSoft
)

// CancelKindForRepaint maps the client's repaint intent to a CancelKind.
// A repaint (vim toggle) is a soft cancel; anything else is hard. This is
// the single point where the wire-level bool becomes a typed decision.
func CancelKindForRepaint(repaint bool) CancelKind {
	if repaint {
		return CancelSoft
	}

	return CancelHard
}

// Repaint reports whether this kind corresponds to a repaint (soft) request.
func (k CancelKind) Repaint() bool {
	return k == CancelSoft
}

func (k CancelKind) String() string {
	switch k {
	case CancelHard:
		return "hard"
	case CancelSoft:
		return "soft"
	default:
		return "unknown"
	}
}
