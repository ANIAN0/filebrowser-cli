package output

import (
	"encoding/json"
	"fmt"
)

// PrintListText writes items as a simple human-readable list.
func (o *Output) PrintListText(items []any) error {
	if len(items) == 0 {
		fmt.Fprintln(o.W, "(no items)")
		return nil
	}
	for i, item := range items {
		b, err := json.Marshal(item)
		if err != nil {
			return err
		}
		fmt.Fprintf(o.W, "%d: %s\n", i+1, string(b))
	}
	return nil
}

// PrintObjectText writes a single object as human-readable text.
func (o *Output) PrintObjectText(obj any) error {
	b, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(o.W, string(b))
	return nil
}

// PrintErrorText writes an error message to the error writer.
func (o *Output) PrintErrorText(err error, code int) error {
	name := ExitCodeNames[code]
	if name == "" {
		name = "error"
	}
	fmt.Fprintf(o.ErrW, "[%s] %v\n", name, err)
	return nil
}