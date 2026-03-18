## ADDED Requirements

### Requirement: Consecutive status poll errors fail the workflow
The workflow SHALL track consecutive GetAgentStatus activity failures. After `maxConsecutiveStatusErrors` (5) consecutive failures, the workflow SHALL transition to Failed phase with a message indicating the sidecar became unreachable. On any successful status poll, the counter SHALL reset to 0.

#### Scenario: Sidecar crashes after agent starts
- **WHEN** GetAgentStatus returns errors for 5 consecutive poll cycles
- **THEN** the workflow transitions to phase "Failed" with message containing "sidecar unreachable"
- **AND** pod cleanup and LLM key revocation execute normally

#### Scenario: Transient network error recovers
- **WHEN** GetAgentStatus fails for 3 consecutive polls then succeeds
- **THEN** the consecutive error counter resets to 0
- **AND** the workflow continues polling normally

#### Scenario: Status poll error is logged
- **WHEN** GetAgentStatus returns an error
- **THEN** the workflow logs a warning with the error message and current consecutive error count

### Requirement: Cleanup errors are logged
The workflow SHALL log errors from RevokeLLMKey and CleanupPod activities via the Temporal workflow logger at Error level. Cleanup errors SHALL NOT change the workflow's final return value.

#### Scenario: Pod cleanup fails
- **WHEN** CleanupPod activity returns an error
- **THEN** the workflow logs the error at Error level with the pod name
- **AND** the workflow still returns nil (does not propagate cleanup error)

#### Scenario: LLM key revocation fails
- **WHEN** RevokeLLMKey activity returns an error
- **THEN** the workflow logs the error at Error level with the key identifier
- **AND** the workflow still returns nil
