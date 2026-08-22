package marshaller

import (
	stdjson "encoding/json"
	"reflect"
	"unicode/utf8"
)

// The backend is held to encoding/json's meaning, not to its spelling.
//
// A difference is worth paying for when it changes what the caller ends up
// with: a field that should have been dropped, a member name that is not even
// valid JSON, an error of a type nobody can match on, a number quietly
// truncated on the way in. Those are diverted to encoding/json, and divert.go
// works them out from the type so the cost lands once per type rather than once
// per call.
//
// A difference that only changes how the same value is written is left alone.
// The backend sorts object members by their escaped form rather than by the
// name, writes a padded exponent in 1e-07 where encoding/json trims it to 1e-7,
// and spells a few escapes the long way. Every one of those decodes back to
// exactly what encoding/json's output decodes to, and catching them meant
// scanning every result, so they are documented instead. See README.md.

// Marshal encodes v as JSON using the backend selected at build time.
func Marshal(v any) ([]byte, error) {
	flags := typeQuirks(reflect.TypeOf(v))
	if flags&quirkMapKey != 0 || (!backendSupportsOmitZero && flags&quirkOmitZero != 0) {
		return stdjson.Marshal(v)
	}

	out, err := backendMarshal(v)
	if err != nil {
		if !backendUsesStdErrors {
			err = conformMarshalError(v, err)
		}
		return nil, err
	}
	return out, nil
}

// Unmarshal decodes JSON into v using the backend selected at build time.
//
// "omitzero" only ever affects encoding, so it does not come into play here.
// Map key types, the "string" tag option and invalid UTF-8 do, and all three
// are settled before the backend is called.
//
// When Unmarshal reports an error the contents of v are unspecified: backends
// differ in how much of a malformed document they apply before giving up.
// encoding/json makes the same reservation for anything past the first error.
func Unmarshal(data []byte, v any) error {
	const divertOnDecode = quirkMapKey | quirkStringOpt
	if typeQuirks(reflect.TypeOf(v))&divertOnDecode != 0 || !utf8.Valid(data) {
		// encoding/json rewrites invalid UTF-8 in decoded strings to U+FFFD
		// and the backend does not, so a document carrying any is decoded by
		// encoding/json outright.
		return stdjson.Unmarshal(data, v)
	}

	err, panicked := guardedBackendUnmarshal(data, v)
	if panicked {
		return stdjson.Unmarshal(data, v)
	}
	if err != nil && !backendUsesStdErrors {
		return conformUnmarshalError(data, v, err)
	}
	return err
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
