package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPrintListJSON_Empty(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriters(ModeJSON, &buf, &buf)

	err := o.PrintListJSON([]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Parse and verify structure
	var result struct {
		Count int   `json:"count"`
		Items []any `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, output)
	}
	if result.Count != 0 {
		t.Errorf("count = %d, want 0", result.Count)
	}
	if result.Items == nil {
		t.Error("items should not be null")
	}
	if len(result.Items) != 0 {
		t.Errorf("items should be empty, got %d", len(result.Items))
	}
}

func TestPrintListJSON_Nil(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriters(ModeJSON, &buf, &buf)

	err := o.PrintListJSON(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	var result struct {
		Count int   `json:"count"`
		Items []any `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, output)
	}
	if result.Count != 0 {
		t.Errorf("count = %d, want 0", result.Count)
	}
	if result.Items == nil {
		t.Error("items should not be null")
	}
}

func TestPrintListJSON_Items(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriters(ModeJSON, &buf, &buf)

	items := []any{
		map[string]any{"name": "alice"},
		map[string]any{"name": "bob"},
	}

	err := o.PrintListJSON(items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	var result struct {
		Count int   `json:"count"`
		Items []any `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, output)
	}
	if result.Count != 2 {
		t.Errorf("count = %d, want 2", result.Count)
	}
	if len(result.Items) != 2 {
		t.Errorf("items length = %d, want 2", len(result.Items))
	}
}

func TestPrintListJSON_FieldOrder(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriters(ModeJSON, &buf, &buf)

	items := []any{map[string]any{"a": 1}}

	err := o.PrintListJSON(items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Check that "count" comes before "items" in the output
	countIdx := strings.Index(output, "\"count\"")
	itemsIdx := strings.Index(output, "\"items\"")
	if countIdx >= itemsIdx {
		t.Errorf("'count' should come before 'items': %s", output)
	}
}

func TestPrintObjectJSON(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriters(ModeJSON, &buf, &buf)

	obj := map[string]any{"key": "value"}

	err := o.PrintObjectJSON(obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, output)
	}
	if result["key"] != "value" {
		t.Errorf("result[key] = %v, want %v", result["key"], "value")
	}
}

func TestPrintErrorJSON(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriters(ModeJSON, &buf, &buf)

	err := o.PrintErrorJSON("test error", ExitClientError)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	var result struct {
		Code  int    `json:"code"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, output)
	}
	if result.Code != ExitClientError {
		t.Errorf("code = %d, want %d", result.Code, ExitClientError)
	}
	if result.Error != "test error" {
		t.Errorf("error = %q, want %q", result.Error, "test error")
	}
}