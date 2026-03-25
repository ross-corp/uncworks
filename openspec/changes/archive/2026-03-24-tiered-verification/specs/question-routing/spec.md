## ADDED Requirements

### Requirement: Manage agent reads implement agent output during review
The system SHALL make the implement agent's structured log (`.aot/logs/agent.jsonl`) available to the manage agent during its review session. The verify activity SHALL read the implement agent's JSONL output and include relevant excerpts in the manage agent's review prompt.

#### Scenario: Implement agent JSONL is read before review prompt construction
- **WHEN** structural pre-screening passes and the manage agent review is about to start
- **THEN** the system reads the implement agent's `.aot/logs/agent.jsonl` from the workspace
- **AND** extracts the last N lines (configurable, default 200) of the log for inclusion in the review prompt

#### Scenario: Implement agent log is unavailable
- **WHEN** the implement agent's JSONL log does not exist or is empty
- **THEN** the manage agent review proceeds without implement agent output excerpts
- **AND** the review prompt notes that no implement agent output was available

### Requirement: Manage agent identifies unanswered questions from implement agent output
The manage agent SHALL be instructed to scan the implement agent's output for unanswered questions — instances where the implement agent expressed uncertainty, asked a question in its output text, or noted that it was guessing due to missing information. The manage agent SHALL either answer the question (if it has enough context) or escalate to the human via `ask_user`.

#### Scenario: Implement agent asked a question in its output
- **WHEN** the implement agent's output contains text like "I'm unsure whether...", "Question:", or "I assumed X but it could be Y"
- **THEN** the manage agent identifies this as an unanswered question in its review
- **AND** either provides an answer (injected into retry feedback) or escalates via `ask_user`

#### Scenario: Implement agent had no questions
- **WHEN** the implement agent's output does not contain any questions or expressions of uncertainty
- **THEN** the manage agent proceeds with its review based solely on the diff and spec scenarios
- **AND** no question routing occurs

### Requirement: Manage agent can escalate questions to human via ask_user
The manage agent SHALL be permitted to call the `ask_user` tool during the verify stage. When it calls `ask_user`, the run enters `waiting_for_input` state. The manage agent synthesizes a well-formed, contextualized question for the human — not raw agent confusion.

#### Scenario: Manage agent escalates a question to the human
- **WHEN** the manage agent determines that a question from the implement agent (or an ambiguity in the spec) requires human input
- **THEN** it calls `ask_user` with a synthesized question that includes context about what the implement agent did and why clarification is needed
- **AND** the run enters `waiting_for_input` state until the human responds

#### Scenario: Human response is incorporated into the review
- **WHEN** the human provides a response to the manage agent's `ask_user` question
- **THEN** the manage agent receives the response and incorporates it into its ongoing review
- **AND** the manage agent either passes the run (if the human's answer confirms the implementation is correct) or triggers a retry with the clarification injected into the feedback

#### Scenario: Manage agent answers the question itself without escalating
- **WHEN** the manage agent reads an implement agent question and determines it can answer the question from context (specs, diff, codebase)
- **THEN** the manage agent does NOT call `ask_user`
- **AND** the answer is included in the review feedback so the implement agent receives it on retry

### Requirement: Question routing creates a chain of command
The implement agent SHALL remain blocked from calling `ask_user` (existing policy). Questions flow upward: implement agent writes questions in output, manage agent reads them during review, manage agent either answers or escalates to human. Humans only see well-formed, contextualized questions synthesized by the manage agent.

#### Scenario: Implement agent cannot call ask_user
- **WHEN** the implement agent attempts to call `ask_user` during the execute stage
- **THEN** the tool call is blocked with the existing policy message
- **AND** the implement agent is instructed to include questions in its output text

#### Scenario: Human sees synthesized questions, not raw agent output
- **WHEN** the manage agent escalates a question to the human
- **THEN** the `ask_user` question text is a synthesized summary written by the manage agent
- **AND** the question includes relevant context (what the implement agent was trying to do, what it was unsure about, what the spec says)
- **AND** the question does NOT simply forward raw implement agent text verbatim
