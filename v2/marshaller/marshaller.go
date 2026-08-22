package marshaller

import (
	"reflect"

	jsonv2 "encoding/json/v2"
)

// Options is encoding/json/v2's option type. It is an alias rather than a type
// of this package's own so that callers can pass any option constructor from
// encoding/json/v2, encoding/json/jsontext or encoding/json directly.
//
// A future backend that is not encoding/json/v2 can still honour these: v2
// exposes GetOption, which reads an option back out of a value and reports
// whether it was set at all, so this package can translate what the backend
// understands and fall back to encoding/json/v2 for the rest.
type Options = jsonv2.Options

// Marshal encodes v as JSON using the backend selected at build time.
func Marshal(v any, opts ...Options) ([]byte, error) {
	return backendMarshal(v, opts...)
}

// Unmarshal decodes JSON into v using the backend selected at build time.
func Unmarshal(data []byte, v any, opts ...Options) error {
	return backendUnmarshal(data, v, opts...)
}

// Preheat compiles the codec for vt ahead of its first use. It is a no-op on
// backends that have nothing to compile.
func Preheat(vt reflect.Type) error {
	return backendPreheat(vt)
}

// PreheatMany is the batch form of Preheat.
func PreheatMany(vts []reflect.Type) error {
	return backendPreheatMany(vts)
}
