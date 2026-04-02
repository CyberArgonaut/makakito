package schema

import (
	"encoding/json"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestDurationYAMLParsing(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"5s", 5 * time.Second},
		{"60s", 60 * time.Second},
		{"1m", 1 * time.Minute},
		{"5m", 5 * time.Minute},
		{"1h", 1 * time.Hour},
		{"1m30s", 90 * time.Second},
		{"500ms", 500 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var d Duration
			if err := yaml.Unmarshal([]byte(tt.input), &d); err != nil {
				t.Fatalf("unmarshal %q: %v", tt.input, err)
			}
			if d.Duration != tt.want {
				t.Errorf("got %v, want %v", d.Duration, tt.want)
			}
		})
	}
}

func TestDurationYAMLInvalid(t *testing.T) {
	invalids := []string{"abc", "5x"}
	for _, input := range invalids {
		t.Run(input, func(t *testing.T) {
			var d Duration
			if err := yaml.Unmarshal([]byte(input), &d); err == nil {
				t.Errorf("expected error for %q, got %v", input, d)
			}
		})
	}
}

func TestDurationJSONRoundTrip(t *testing.T) {
	d := Duration{5 * time.Minute}
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Duration
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Duration != d.Duration {
		t.Errorf("got %v, want %v", got.Duration, d.Duration)
	}
}
