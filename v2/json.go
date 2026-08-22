// Package json delivers encoding/json/v2 semantics behind an import path this
// module controls.
//
// Every call here forwards to encoding/json/v2 as it stands. The point of the
// indirection is the seam: when a backend appears that implements v2 semantics
// faster than the standard library, it slots in behind these functions and
// callers pick it up by updating the dependency, with no change to their code.
package json

import (
	"reflect"

	"go.gh.ink/json/v2/marshaller"
)

// Options configures a call. It is encoding/json/v2's own option type, so every
// option constructor in encoding/json/v2, encoding/json/jsontext and
// encoding/json applies here.
//
// Passing encoding/json.DefaultOptionsV1() runs a call with v1 semantics on the
// v2 engine, which is useful while migrating a caller that still has consumers
// pinned to v1 output.
type Options = marshaller.Options

// Marshal encodes v as JSON.
func Marshal(v any, opts ...Options) ([]byte, error) {
	return marshaller.Marshal(v, opts...)
}

// Unmarshal decodes JSON into v.
func Unmarshal(data []byte, v any, opts ...Options) error {
	return marshaller.Unmarshal(data, v, opts...)
}

// Preheat compiles the codec for vt ahead of its first use. It is a no-op on
// backends that have nothing to compile, which today means all of them.
//
// It exists so that callers written against this package keep working when a
// backend that does precompile arrives.
func Preheat(vt reflect.Type) error {
	return marshaller.Preheat(vt)
}

// PreheatMany is the batch form of Preheat.
func PreheatMany(vts []reflect.Type) error {
	return marshaller.PreheatMany(vts)
}
