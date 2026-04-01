package experiment

import (
	"testing"
)

func TestLoadValidFile(t *testing.T) {
	exp, err := Load("testdata/valid_experiment.yaml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if exp.Name != "Redis blackout" {
		t.Errorf("name = %q, want %q", exp.Name, "Redis blackout")
	}
	if exp.Version != "1" {
		t.Errorf("version = %q, want %q", exp.Version, "1")
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("testdata/nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	_, err := Parse([]byte(":\n  invalid: [yaml"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadBadDuration(t *testing.T) {
	yaml := `version: "1"
name: test
hypothesis:
  title: t
  probes:
    - type: http
      url: http://x
      expected_status: 200
method:
  - type: fault
    name: n
    target:
      kind: docker
    fault:
      kind: cpu
    duration: notaduration
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
}
