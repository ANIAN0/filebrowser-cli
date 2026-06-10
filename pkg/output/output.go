package output

import (
	"io"
	"os"
)

// Mode constants for output formatting.
const (
	ModeText = "text"
	ModeJSON = "json"
)

// Output handles formatted output for CLI tools.
type Output struct {
	// Mode is the output mode: "text" or "json".
	Mode string

	// W is the writer for normal output (default: os.Stdout).
	W io.Writer

	// ErrW is the writer for error output (default: os.Stderr).
	ErrW io.Writer
}

// New creates a new Output with the given mode and default writers.
func New(mode string) *Output {
	return &Output{
		Mode: mode,
		W:    os.Stdout,
		ErrW: os.Stderr,
	}
}

// NewWithWriters creates a new Output with custom writers.
func NewWithWriters(mode string, w, errW io.Writer) *Output {
	return &Output{
		Mode: mode,
		W:    w,
		ErrW: errW,
	}
}

// PrintList prints a list of items in the configured mode.
func (o *Output) PrintList(items []any) error {
	if o.Mode == ModeJSON {
		return o.PrintListJSON(items)
	}
	return o.PrintListText(items)
}

// PrintObject prints a single object in the configured mode.
func (o *Output) PrintObject(obj any) error {
	if o.Mode == ModeJSON {
		return o.PrintObjectJSON(obj)
	}
	return o.PrintObjectText(obj)
}

// PrintError prints an error message in the configured mode.
func (o *Output) PrintError(err error, code int) error {
	if o.Mode == ModeJSON {
		return o.PrintErrorJSON(err.Error(), code)
	}
	return o.PrintErrorText(err, code)
}

// ExitWithError prints the error and exits with the given code.
func (o *Output) ExitWithError(err error, code int) {
	if o.Mode == ModeJSON {
		o.PrintErrorJSON(err.Error(), code)
	} else {
		o.PrintErrorText(err, code)
	}
	os.Exit(code)
}