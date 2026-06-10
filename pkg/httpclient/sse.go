package httpclient

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

// ParseSSEStream reads SSE-formatted data lines and aggregates them into a slice.
//
// SSE format:
//
//	data: {"key": "value"}
//
//	data: {"key": "value2"}
//
// Only "data:" lines are parsed. Other lines (event:, id:, comments) are ignored.
// Empty data lines are skipped.
func ParseSSEStream(r io.Reader) ([]map[string]any, error) {
	var results []map[string]any
	scanner := bufio.NewScanner(r)

	// Increase buffer size for large SSE payloads
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB max line

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// Skip non-data lines
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		// Extract payload after "data:"
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}

		// Parse JSON
		var m map[string]any
		if err := json.Unmarshal([]byte(payload), &m); err != nil {
			return results, err
		}
		results = append(results, m)
	}

	return results, scanner.Err()
}

// ParseSSELines is a simpler variant that takes a string and returns parsed JSON objects.
func ParseSSELines(s string) ([]map[string]any, error) {
	return ParseSSEStream(strings.NewReader(s))
}