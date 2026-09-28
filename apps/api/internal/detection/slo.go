package detection

// SLO is a local-window calculation. It is not a 30-day reliability claim.
type SLO struct {
	Target              float64
	Availability        float64
	ErrorBudgetConsumed float64
	BurnRate            float64
	Sufficient          bool
	Window              string
}

func ComputeSLO(requests, errors float64) SLO {
	out := SLO{Target: SLOTarget, Window: "15m"}
	if requests < MinRequests {
		return out
	}
	availability := 1 - errors/requests
	budget := 1 - SLOTarget
	consumed := 0.0
	burn := 0.0
	if budget > 0 {
		consumed = (1 - availability) / budget
		burn = (errors / requests) / budget
	}
	out.Availability = availability
	out.ErrorBudgetConsumed = consumed
	out.BurnRate = burn
	out.Sufficient = true
	return out
}
