## Purpose

Define the chat-based run creation flow with optional AI refinement before launch.

## ADDED Requirements

### Requirement: Quick prompt input as default
The new run view SHALL present a text input for the prompt as the primary input, with repo selection and a "Run" button.

#### Scenario: Quick run from prompt
- **WHEN** the user types a prompt and clicks Run
- **THEN** an agent run is created immediately with the prompt

### Requirement: Optional AI refinement via chat
The new run view SHALL offer a "Refine with AI" option that opens a chat conversation to iterate on the spec before launching.

#### Scenario: Refine with AI
- **WHEN** the user clicks "Refine with AI"
- **THEN** the prompt is sent to the LLM for planning feedback
- **AND** the response appears as a chat message with the proposed spec
- **AND** the user can reply to refine further or click "Run Agent" to launch

### Requirement: Spec tab for full specifications
The new run view SHALL offer a Spec tab that swaps the prompt input for a Monaco editor with markdown highlighting.

#### Scenario: Switch to spec mode
- **WHEN** the user clicks the Spec tab
- **THEN** the text input is replaced with a Monaco editor
- **AND** the orchestration mode auto-switches to spec-driven

### Requirement: Config collapsed by default
Model, TTL, and orchestration mode SHALL be shown as a single collapsed line of defaults, expandable on click.

#### Scenario: Collapsed config
- **WHEN** the new run view loads
- **THEN** config is shown as "qwen3:8b · 15m · single" with an expand button
