package fault

import "time"

// State is the in-process fault switch for payment-api.
// It expires on its own. Nothing else in the cluster is affected.
type State struct {
	enabled bool
	until   time.Time
	// ErrorPercent of requests return HTTP 500 while enabled.
	ErrorPercent int
	Latency      time.Duration
}

func (s *State) Enable(until time.Time, errorPercent int, latency time.Duration) {
	if errorPercent < 0 {
		errorPercent = 0
	}
	if errorPercent > 100 {
		errorPercent = 100
	}
	if latency < 0 {
		latency = 0
	}
	s.enabled = true
	s.until = until
	s.ErrorPercent = errorPercent
	s.Latency = latency
}

func (s *State) Disable() {
	s.enabled = false
	s.until = time.Time{}
}

func (s *State) Active(now time.Time) bool {
	return s.enabled && now.Before(s.until)
}

func (s *State) Until() time.Time {
	return s.until
}
