package server

import "expvar"

var (
	// RunsCreatedTotal counts the total number of agent runs created.
	RunsCreatedTotal = expvar.NewInt("uncworks_runs_created_total")
	// RunsSucceededTotal counts the total number of agent runs that succeeded.
	RunsSucceededTotal = expvar.NewInt("uncworks_runs_succeeded_total")
	// RunsFailedTotal counts the total number of agent runs that failed.
	RunsFailedTotal = expvar.NewInt("uncworks_runs_failed_total")
	// RunsCancelledTotal counts the total number of agent runs that were cancelled.
	RunsCancelledTotal = expvar.NewInt("uncworks_runs_cancelled_total")
)