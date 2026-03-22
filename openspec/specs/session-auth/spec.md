# session-auth Specification

## Purpose
TBD - created by archiving change backend-for-frontend. Update Purpose after archive.
## Requirements
### Requirement: Cookie-based session management
The system SHALL support cookie-based sessions with CSRF protection, replacing per-request API key headers.

#### Scenario: Session creation
- **WHEN** a user authenticates (via API key or OIDC)
- **THEN** the BFF SHALL create an HttpOnly, Secure, SameSite=Strict session cookie

#### Scenario: CSRF protection
- **WHEN** a state-changing request (POST/PUT/DELETE) is received
- **THEN** the BFF SHALL verify a CSRF token in the request header matches the session

#### Scenario: Open mode for local dev
- **WHEN** the BFF is configured with `AUTH_MODE=open`
- **THEN** all requests SHALL be allowed without authentication

### Requirement: Rate limiting per session
The system SHALL enforce per-session rate limits to prevent API abuse.

#### Scenario: Rate limit exceeded
- **WHEN** a session exceeds 100 requests per second
- **THEN** the BFF SHALL return HTTP 429 with a Retry-After header

