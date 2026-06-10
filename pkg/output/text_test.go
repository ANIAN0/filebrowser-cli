package output

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestPrintListText_Empty(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriters(ModeText, &buf, &buf)

	err := o.PrintListText([]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "(no items)") {
		t.Errorf("output should contain '(no items)': %s", output)
	}
}

func TestPrintListText_Nil(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriters(ModeText, &buf, &buf)

	err := o.PrintListText(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "(no items)") {
		t.Errorf("output should contain '(no items)': %s", output)
	}
}

func TestPrintListText_Items(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriters(ModeText, &buf, &buf)

	items := []any{
		map[string]any{"name": "alice"},
		map[string]any{"name": "bob"},
	}

	err := o.PrintListText(items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d: %s", len(lines), output)
	}
}

func TestPrintObjectText(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriters(ModeText, &buf, &buf)

	obj := map[string]any{"key": "value"}

	err := o.PrintObjectText(obj)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "key") || !strings.Contains(output, "value") {
		t.Errorf("output should contain 'key' and 'value': %s", output)
	}
}

func TestPrintErrorText(t *testing.T) {
	var buf bytes.Buffer
	o := NewWithWriters(ModeText, &buf, &buf)

	err := o.PrintErrorText(errors.New("test error"), ExitClientError)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "test error") {
		t.Errorf("output should contain 'test error': %s", output)
	}
	if !strings.Contains(output, "[client error]") {
		t.Errorf("output should contain '[client error]': %s", output)
	}
}