package httpclient

import (
	"strings"
	"testing"
)

func TestParseSSEStream_Basic(t *testing.T) {
	input := `data: {"key": "value1"}

data: {"key": "value2"}

data: {"key": "value3"}

`
	results, err := ParseSSELines(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0]["key"] != "value1" {
		t.Errorf("results[0][key] = %v, want %v", results[0]["key"], "value1")
	}
	if results[1]["key"] != "value2" {
		t.Errorf("results[1][key] = %v, want %v", results[1]["key"], "value2")
	}
	if results[2]["key"] != "value3" {
		t.Errorf("results[2][key] = %v, want %v", results[2]["key"], "value3")
	}
}

func TestParseSSEStream_Empty(t *testing.T) {
	results, err := ParseSSELines("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestParseSSEStream_NoDataLines(t *testing.T) {
	input := `event: message
id: 1
retry: 5000

event: message
id: 2
`
	results, err := ParseSSELines(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestParseSSEStream_MixedLines(t *testing.T) {
	input := `event: message
data: {"key": "value1"}

id: 1
data: {"key": "value2"}

: this is a comment
data: {"key": "value3"}

`
	results, err := ParseSSELines(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
}

func TestParseSSEStream_EmptyDataLine(t *testing.T) {
	input := `data:

data: {"key": "value"}

data:

`
	results, err := ParseSSELines(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestParseSSEStream_InvalidJSON(t *testing.T) {
	input := `data: {invalid json}

data: {"key": "value"}

`
	_, err := ParseSSELines(input)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseSSEStream_ComplexJSON(t *testing.T) {
	input := `data: {"name": "test", "count": 42, "nested": {"a": 1, "b": "two"}}

`
	results, err := ParseSSELines(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0]["name"] != "test" {
		t.Errorf("name = %v, want %v", results[0]["name"], "test")
	}
	if results[0]["count"] != float64(42) {
		t.Errorf("count = %v, want %v", results[0]["count"], float64(42))
	}
}

func TestParseSSEStream_LargePayload(t *testing.T) {
	// Create a large payload
	largeValue := strings.Repeat("x", 100000)
	input := `data: {"key": "` + largeValue + `"}

`
	results, err := ParseSSELines(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0]["key"] != largeValue {
		t.Error("large payload not parsed correctly")
	}
}

func TestParseSSEStream_Reader(t *testing.T) {
	input := `data: {"key": "value1"}

data: {"key": "value2"}

`
	reader := strings.NewReader(input)
	results, err := ParseSSEStream(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}