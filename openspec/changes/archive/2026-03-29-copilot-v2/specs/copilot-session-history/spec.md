## ADDED Requirements

### Requirement: Session persistence
Chat sessions MUST be persisted to localStorage and survive page refreshes and navigation.

#### Scenario: Session created on first message
- **WHEN** user sends the first message in a new session
- **THEN** a session is created with the first user message as title (truncated to 40 chars)

#### Scenario: Session restored on reload
- **WHEN** user refreshes the page
- **THEN** the last active session's messages are shown in the panel

#### Scenario: New chat
- **WHEN** user clicks "New chat" in the panel header
- **THEN** a new empty session is started, previous session is saved

#### Scenario: Session list
- **WHEN** user opens the session dropdown in the panel header
- **THEN** up to 20 recent sessions are listed by title and relative time

#### Scenario: Session pruning
- **WHEN** more than 20 sessions exist
- **THEN** the oldest session is removed from localStorage
