package output

import (
	"encoding/json"
	"fmt"
)

// PrintListJSON writes items wrapped in { "items": [...], "count": N }.
// Stable field order: count first, items second.
// Empty lists return {"count":0,"items":[]} not null.
func (o *Output) PrintListJSON(items []any) error {
	if items == nil {
		items = []any{} // never null
	}
	wrapper := struct {
		Count int   `json:"count"`
		Items []any `json:"items"`
	}{
		Count: len(items),
		Items: items,
	}
	b, err := json.Marshal(wrapper)
	if err != nil {
		return err
	}
	fmt.Fprintln(o.W, string(b))
	return nil
}

// PrintObjectJSON writes a single object directly without wrapping.
func (o *Output) PrintObjectJSON(obj any) error {
	b, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	fmt.Fprintln(o.W, string(b))
	return nil
}

// PrintErrorJSON writes { "error": msg, "code": N }.
func (o *Output) PrintErrorJSON(msg string, code int) error {
	wrapper := struct {
		Code  int    `json:"code"`
		Error string `json:"error"`
	}{
		Code:  code,
		Error: msg,
	}
	b, err := json.Marshal(wrapper)
	if err != nil {
		return err
	}
	fmt.Fprintln(o.W, string(b))
	return nil
}