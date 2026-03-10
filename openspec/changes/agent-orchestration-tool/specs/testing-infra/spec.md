## ADDED Requirements

### Requirement: Zero-Regression Testing Mandate
Every feature, bug fix, or requirement change SHALL include a corresponding suite of unit, integration, and E2E tests.

#### Scenario: Submitting a change without tests
- **WHEN** a change is submitted to the Control Plane
- **THEN** it SHALL be rejected if the test coverage for the new code is below the 100% threshold

### Requirement: Playwright E2E Integration
The WebUI SHALL be verified using **Playwright** against a real k0s cluster.

#### Scenario: Full system E2E verification
- **WHEN** the Playwright suite is executed
- **THEN** it SHALL spin up a real k0s cluster, deploy the Orchestrator, and verify the full trace of an agent fixed PR

### Requirement: gRPC Contract Testing
The system SHALL use gRPC contract tests to verify the interface between the Control Plane and the `pi-mono` sidecars.

#### Scenario: Detecting breaking gRPC changes
- **WHEN** a change is made to the `Agent.proto` file
- **THEN** the contract tests SHALL fail if the new definition breaks backward compatibility with existing sidecars
