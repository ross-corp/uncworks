package main

import (
	"strings"
	"testing"
	"time"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
)

func TestParseSinceDuration(t *testing.T) {
	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"1h", time.Hour, false},
		{"24h", 24 * time.Hour, false},
		{"30m", 30 * time.Minute, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"0d", 0, true},
		{"-1d", 0, true},
		{"notaduration", 0, true},
		{"", 0, true},
	}
	for _, tt := range tests {
		d, err := parseSinceDuration(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseSinceDuration(%q): expected error, got nil (duration=%v)", tt.input, d)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSinceDuration(%q): unexpected error: %v", tt.input, err)
			continue
		}
		if d != tt.want {
			t.Errorf("parseSinceDuration(%q) = %v, want %v", tt.input, d, tt.want)
		}
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		t      time.Time
		suffix string
	}{
		{now.Add(-30 * time.Second), "s ago"},
		{now.Add(-5 * time.Minute), "m ago"},
		{now.Add(-3 * time.Hour), "h ago"},
		{now.Add(-2 * 24 * time.Hour), "d ago"},
	}
	for _, tt := range tests {
		got := relativeTime(tt.t)
		if !strings.HasSuffix(got, tt.suffix) {
			t.Errorf("relativeTime(%v) = %q, want suffix %q", tt.t, got, tt.suffix)
		}
	}
}

func TestPhaseLabel(t *testing.T) {
	tests := []struct {
		input int32
		want  string
	}{
		{0, "UNKNOWN"},  // UNSPECIFIED
		{1, "PENDING"},
		{2, "RUNNING"},
		{3, "WAITING"},  // WAITING_FOR_INPUT
		{4, "DONE"},     // SUCCEEDED
		{5, "FAILED"},
		{6, "CANCELLED"},
	}
	for _, tt := range tests {
		got := phaseLabel(apiv1.AgentRunPhase(tt.input))
		if got != tt.want {
			t.Errorf("phaseLabel(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
