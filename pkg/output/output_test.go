package output

import (
	"bytes"
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	o := New(ModeText)
	if o.Mode != ModeText {
		t.Errorf("Mode = %q, want %q", o.Mode, ModeText)
	}
	if o.W == nil {
		t.Error("W should not be nil")
	}
	if o.ErrW == nil {
		t.Error("ErrW should not be nil")
	}
}

func TestNewWithWriters(t *testing.T) {
	var buf bytes.Buffer
	var errBuf bytes.Buffer
	o := NewWithWriters(ModeJSON, &buf, &errBuf)
	if o.Mode != ModeJSON {
		t.Errorf("Mode = %q, want %q", o.Mode, ModeJSON)
	}
	if o.W != &buf {
		t.Error("W should be &buf")
	}
	if o.ErrW != &errBuf {
		t.Error("ErrW should be &errBuf")
	}
}

func TestPrintList_Text(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriters(ModeText, &buf, &buf)

	items := []any{
		map[string]any{"a": 1},
		map[string]any{"a": 2},
	}

	err := o.PrintList(items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("output should not be empty")
	}
}

func TestPrintList_JSON(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriters(ModeJSON, &buf, &buf)

	items := []any{
		map[string]any{"a": 1},
		map[string]any{"a": 2},
	}

	err := o.PrintList(items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("output should not be empty")
	}
	// Should contain "items" and "count"
	if !contains(output, "items") || !contains(output, "count") {
		t.Errorf("output should contain 'items' and 'count': %s", output)
	}
}

func TestPrintObject_Text(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriters(ModeText, &buf, &buf)

	obj := map[string]any{"key": "value"}

	err := o.PrintObject(obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("output should not be empty")
	}
}

func TestPrintObject_JSON(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriters(ModeJSON, &buf, &buf)

	obj := map[string]any{"key": "value"}

	err := o.PrintObject(obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("output should not be empty")
	}
	if !contains(output, "key") || !contains(output, "value") {
		t.Errorf("output should contain 'key' and 'value': %s", output)
	}
}

func TestPrintError_Text(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriters(ModeText, &buf, &buf)

	err := errors.New("test error")
	err = o.PrintError(err, ExitClientError)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("output should not be empty")
	}
}

func TestPrintError_JSON(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriters(ModeJSON, &buf, &buf)

	err := errors.New("test error")
	err = o.PrintError(err, ExitClientError)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("output should not be empty")
	}
	if !contains(output, "error") || !contains(output, "code") {
		t.Errorf("output should contain 'error' and 'code': %s", output)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}