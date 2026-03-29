## ADDED Requirements

### Requirement: Webhook endpoint requires HMAC secret to be configured
The webhook handler SHALL reject all incoming webhook requests with HTTP 401 when no `WebhookSecret` is configured on the server. It MUST NOT process the webhook payload if signature verification cannot be performed.

#### Scenario: Webhook rejected when secret not configured
- **WHEN** the server starts with an empty `WebhookSecret`
- **WHEN** a POST request arrives at the webhook endpoint
- **THEN** the server returns HTTP 401
- **THEN** no run is triggered

#### Scenario: Webhook accepted with valid HMAC signature
- **WHEN** `WebhookSecret` is configured
- **WHEN** a POST request arrives with a valid HMAC-SHA256 `X-Hub-Signature-256` header
- **THEN** the server processes the request normally

#### Scenario: Webhook rejected with invalid HMAC signature
- **WHEN** `WebhookSecret` is configured
- **WHEN** a POST request arrives with an invalid or missing signature header
- **THEN** the server returns HTTP 401
