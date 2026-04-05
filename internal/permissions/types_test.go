package permissions

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPatternOrArray_UnmarshalYAML_String(t *testing.T) {
	input := `"--verbose"`
	var p PatternOrArray
	if err := yaml.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Values) != 1 || p.Values[0] != "--verbose" {
		t.Fatalf("expected [--verbose], got %v", p.Values)
	}
}

func TestPatternOrArray_UnmarshalYAML_Array(t *testing.T) {
	input := "- --foo\n- --bar"
	var p PatternOrArray
	if err := yaml.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Values) != 2 || p.Values[0] != "--foo" || p.Values[1] != "--bar" {
		t.Fatalf("expected [--foo --bar], got %v", p.Values)
	}
}

func TestPatternOrArray_UnmarshalYAML_Invalid(t *testing.T) {
	input := "42"
	var p PatternOrArray
	err := yaml.Unmarshal([]byte(input), &p)
	if err == nil {
		t.Fatal("expected error for integer input, got nil")
	}
}

func TestPatternOrArray_UnmarshalJSON_String(t *testing.T) {
	input := `"--verbose"`
	var p PatternOrArray
	if err := json.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Values) != 1 || p.Values[0] != "--verbose" {
		t.Fatalf("expected [--verbose], got %v", p.Values)
	}
}

func TestPatternOrArray_UnmarshalJSON_Array(t *testing.T) {
	input := `["--foo", "--bar"]`
	var p PatternOrArray
	if err := json.Unmarshal([]byte(input), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Values) != 2 || p.Values[0] != "--foo" || p.Values[1] != "--bar" {
		t.Fatalf("expected [--foo --bar], got %v", p.Values)
	}
}

func TestPatternOrArray_UnmarshalJSON_Invalid(t *testing.T) {
	input := `42`
	var p PatternOrArray
	err := json.Unmarshal([]byte(input), &p)
	if err == nil {
		t.Fatal("expected error for integer input, got nil")
	}
}
