// test/testutil/webhook.go — Shared GitHub webhook test helpers.
// signWebhookPayload and buildWebhookPushPayload were private to
// test/regression/webhook_delivery_test.go; exporting them here makes them
// available to any future webhook-related test package without duplication.
package testutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// SignWebhookPayload computes the GitHub-style HMAC-SHA256 signature for a
// webhook payload.  The returned string has the "sha256=" prefix expected by
// the X-Hub-Signature-256 header.
func SignWebhookPayload(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// BuildWebhookPushPayload builds a minimal GitHub push event JSON payload
// that the WebhookHandler can parse.  repo is the full repository name
// (e.g. "org/repo") and addedFiles lists the files added in the push.
func BuildWebhookPushPayload(repo string, addedFiles []string) []byte {
	type commit struct {
		Added    []string `json:"added"`
		Modified []string `json:"modified"`
	}
	type repoInfo struct {
		FullName string `json:"full_name"`
	}
	type payload struct {
		Ref        string   `json:"ref"`
		After      string   `json:"after"`
		Repository repoInfo `json:"repository"`
		Commits    []commit `json:"commits"`
	}
	p := payload{
		Ref:        "refs/heads/main",
		After:      "abc123",
		Repository: repoInfo{FullName: repo},
		Commits:    []commit{{Added: addedFiles, Modified: []string{}}},
	}
	data, _ := json.Marshal(p)
	return data
}
