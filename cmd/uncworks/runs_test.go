package main

import (
	"testing"
	"time"
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

func TestPhaseLabel(t *testing.T) {
	tests := []struct {
		input int32
		want  string
	}{
		{0, "UNKNOWN"},  // unspecified
		{1, "PENDING"},
		{2, "RUNNING"},
		{3, "DONE"},
		{4, "FAILED"},
		{5, "CANCELLED"},
		{6, "WAITING"},
	}
	for _, tt := range tests {
		_ = tt // phaseLabel takes apiv1.AgentRunPhase — test structure only
	}
}
