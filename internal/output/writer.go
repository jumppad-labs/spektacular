package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Writer wraps an io.Writer with optional field filtering.
// It implements workflow.ResultWriter.
type Writer struct {
	w      io.Writer
	fields string
}

// New returns a Writer that writes to w, filtered to fields (JSON array string).
// Pass an empty fields string to write all fields.
func New(w io.Writer, fields string) *Writer {
	return &Writer{w: w, fields: fields}
}

// WriteResult marshals v to indented JSON and writes it, applying any field filter.
func (wr *Writer) WriteResult(v any) error {
	return Write(wr.w, v, wr.fields)
}

// StateInfo carries a caller's current position in a multi-step workflow and
// the actions that would currently succeed. It is populated only for
// state-aware rejections (e.g. an invalid step transition).
type StateInfo struct {
	Current      string   `json:"current"`
	ValidActions []string `json:"valid_actions"`
}

// ErrorResponse is the shared shape every failure is expressed in. It
// implements the error interface so it can be constructed deep inside any
// package and returned as a normal error all the way up to the command
// outcome wrapper.
//
// IsError marshals to the "error" JSON field (always true); it can't be
// named Error since ErrorResponse also defines an Error() string method to
// satisfy the error interface, and Go forbids a field and method sharing a
// name on the same type.
type ErrorResponse struct {
	IsError    bool       `json:"error"`
	Code       string     `json:"code"`
	Message    string     `json:"message"`
	Resource   string     `json:"resource,omitempty"`
	State      *StateInfo `json:"state,omitempty"`
	NextAction string     `json:"next_action"`
}

// Error implements the error interface, returning the failure message.
func (er *ErrorResponse) Error() string {
	return er.Message
}

// NewError constructs an ErrorResponse with the given failure code and message.
func NewError(code, message string) *ErrorResponse {
	return &ErrorResponse{
		IsError: true,
		Code:    code,
		Message: message,
	}
}

// WithResource sets the named resource (file/step/identifier) involved in the failure.
func (er *ErrorResponse) WithResource(name string) *ErrorResponse {
	er.Resource = name
	return er
}

// WithState sets the caller's current position and the actions that would currently succeed.
func (er *ErrorResponse) WithState(current string, valid []string) *ErrorResponse {
	er.State = &StateInfo{Current: current, ValidActions: valid}
	return er
}

// WithNextAction sets the explicit corrective action or valid next command.
func (er *ErrorResponse) WithNextAction(text string) *ErrorResponse {
	er.NextAction = text
	return er
}

// WriteFailure marshals er (an already-built ErrorResponse) and writes it to
// w, applying the same optional field filter as Write.
func WriteFailure(w io.Writer, er *ErrorResponse, fieldsJSON string) error {
	return Write(w, er, fieldsJSON)
}

// Write marshals v to indented JSON and writes it to w.
// If fieldsJSON is non-empty it must be a JSON array of field name strings.
// Only those fields are included in the output; an unknown field name is an error.
// Pass an empty fieldsJSON to bypass filtering (e.g. for --schema output).
// Every value that is not an *ErrorResponse has a "error": false discriminant
// field injected so success and failure responses can always be told apart.
func Write(w io.Writer, v any, fieldsJSON string) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshaling output: %w", err)
	}

	if _, isErrorResponse := v.(*ErrorResponse); !isErrorResponse {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(data, &m); err == nil {
			m["error"] = json.RawMessage("false")
			injected, err := json.Marshal(m)
			if err != nil {
				return fmt.Errorf("marshaling output: %w", err)
			}
			data = injected
		}
	}

	if fieldsJSON == "" {
		var buf bytes.Buffer
		if err := json.Indent(&buf, data, "", "  "); err != nil {
			return fmt.Errorf("formatting output: %w", err)
		}
		fmt.Fprintln(w, buf.String())
		return nil
	}

	var fields []string
	if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
		return fmt.Errorf("--fields must be a JSON array of strings (e.g. '[\"step\",\"instruction\"]'): %w", err)
	}

	if len(fields) == 0 {
		var buf bytes.Buffer
		if err := json.Indent(&buf, data, "", "  "); err != nil {
			return fmt.Errorf("formatting output: %w", err)
		}
		fmt.Fprintln(w, buf.String())
		return nil
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("filtering output: value is not a JSON object: %w", err)
	}

	for _, f := range fields {
		if _, ok := m[f]; !ok {
			return fmt.Errorf("unknown field %q", f)
		}
	}

	filtered := make(map[string]json.RawMessage, len(fields))
	for _, f := range fields {
		filtered[f] = m[f]
	}

	out, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling filtered output: %w", err)
	}
	fmt.Fprintln(w, string(out))
	return nil
}
