## ADDED Requirements

### Requirement: Approval mode badge in unified run list
RunListView's unified run rows SHALL display the approval mode as a secondary badge alongside the kind badge.

#### Scenario: HITL run shows HITL badge
- **WHEN** a unified run row is rendered for an agent run with `approvalMode` equal to `"hitl"`
- **THEN** a badge with label "hitl" is shown next to the kind badge

#### Scenario: LLM-judge run shows badge
- **WHEN** a unified run row is rendered for an agent run with `approvalMode` equal to `"llm-judge"` or `"hybrid"` or an empty string (default)
- **THEN** a badge with label "llm-judge" is shown next to the kind badge

#### Scenario: Chain runs show no approval-mode badge
- **WHEN** a unified run row is rendered for a chain run (not an agent run)
- **THEN** no approval-mode badge is shown
