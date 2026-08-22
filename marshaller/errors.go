package marshaller

import (
	stdjson "encoding/json"
	"reflect"
)

// A backend is free to report failures with its own error types. Downstream
// code routinely inspects them, for example turning a decode failure into a
// 400 response:
//
//	var typeErr *json.UnmarshalTypeError
//	if errors.As(err, &typeErr) { ... }
//
// so an error whose concrete type is not encoding/json's silently defeats that
// check. The functions below restate such an error by replaying the same call
// through encoding/json and returning its error instead.
//
// Replaying is preferred over translating field by field. encoding/json's
// SyntaxError keeps its message unexported, so a translated one would arrive
// blank, and the backend does not always classify a failure the way
// encoding/json does or fill the same descriptive fields. Replaying reproduces
// the type, the message and the offset exactly, and only failures pay for it.

// conformMarshalError returns the error encoding/json reports for v, falling
// back to err if encoding/json accepts a value the backend rejected.
func conformMarshalError(v any, err error) error {
	if _, stdErr := stdjson.Marshal(v); stdErr != nil {
		return stdErr
	}
	return err
}

// conformUnmarshalError returns the error encoding/json reports for the same
// input, falling back to err if encoding/json accepts what the backend
// rejected.
//
// The replay decodes into a scratch value rather than v: v is the caller's, and
// the failed call has already left it in an unspecified state.
func conformUnmarshalError(data []byte, v any, err error) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		// encoding/json rejects these before reading the input, so handing it
		// v directly is safe and yields the InvalidUnmarshalError it would
		// have produced.
		if stdErr := stdjson.Unmarshal(data, v); stdErr != nil {
			return stdErr
		}
		return err
	}
	scratch := reflect.New(rv.Type().Elem())
	if stdErr := stdjson.Unmarshal(data, scratch.Interface()); stdErr != nil {
		return stdErr
	}
	return err
}

// guardedBackendUnmarshal runs the backend and reports a panic instead of
// letting it out.
//
// goccy/go-json reads past the end of its buffer decoding an object member
// name: internal/decoder/struct.go indexes buf[cursor] in
// decodeKeyCharByEscapedChar with no bounds check, so a truncated escape in a
// name can bring the process down. Unmarshal is exactly where untrusted bytes
// arrive, so that is contained here and the document handed to encoding/json,
// which rejects it properly.
//
// A panic raised by the caller's own UnmarshalJSON is caught too, but the
// replay through encoding/json calls the same method and raises it again, so it
// still reaches the caller.
func guardedBackendUnmarshal(data []byte, v any) (err error, panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	return backendUnmarshal(data, v), false
}
