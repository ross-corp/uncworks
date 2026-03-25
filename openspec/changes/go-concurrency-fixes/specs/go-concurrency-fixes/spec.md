## ADDED Requirements

### Requirement: WebSocket exec handler goroutine lifecycle
All goroutines launched within HandleExec SHALL be tracked by the WaitGroup such that wg.Wait() does not return until every goroutine has exited.

#### Scenario: Handler waits for WebSocket reader to exit
- **WHEN** the SPDY stream ends and context is cancelled
- **THEN** HandleExec SHALL NOT return until the WebSocket reader goroutine has fully exited

### Requirement: AgentProcess state access is thread-safe
Reads and writes of AgentProcess.state and AgentProcess.exitError SHALL be performed while holding the AgentProcess mutex.

#### Scenario: Concurrent state read from stream goroutine
- **WHEN** GetAgentStatus or a stream goroutine reads AgentProcess.state concurrently with a state transition
- **THEN** the read SHALL observe a consistent value with no data race

### Requirement: Controller status update errors trigger requeue
Status update errors returned by r.Status().Update SHALL be propagated to the controller-runtime reconcile loop so that transient failures are retried automatically.

#### Scenario: Transient API server error on status update
- **WHEN** r.Status().Update returns a non-nil error
- **THEN** the controller SHALL return that error to trigger an exponential-backoff requeue

### Requirement: BFF middleware maps are bounded in size
Session, CSRF token, and rate-limit IP bucket maps SHALL evict entries that have not been accessed within their TTL to prevent unbounded memory growth.

#### Scenario: Session map eviction
- **WHEN** a session entry has existed longer than the session TTL
- **THEN** the entry SHALL be removed from the in-memory session map by the next eviction sweep

#### Scenario: IP bucket eviction
- **WHEN** an IP bucket has not received a request for more than 10 minutes
- **THEN** the bucket SHALL be removed from the rate-limit map by the next eviction sweep

### Requirement: CI autofix timer callback respects server shutdown
The timer callback that calls createFixRun SHALL use the server-lifetime context rather than context.Background(), and SHALL NOT execute the fix run if the server context has been cancelled.

#### Scenario: Server shutdown during pending timer
- **WHEN** the server context is cancelled while a 30-second fix timer is pending
- **THEN** the timer callback SHALL detect context cancellation and skip calling createFixRun
