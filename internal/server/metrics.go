package server

import "expvar"

var (
	// runsCreatedTotal counts the total number of agent runs created.
	runsCreatedTotal = expvar.NewInt("uncworks_runs_created_total")
	// runsSucceededTotal counts the total number of agent runs that succeeded.
	runsSucceededTotal = expvar.NewInt("uncworks_runs_succeeded_total")
	// runsFailedTotal counts the total number of agent runs that failed.
	runsFailedTotal = expvar.NewInt("uncworks_runs_failed_total")
)