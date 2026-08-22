package marshaller

import (
	"reflect"

	"github.com/goccy/go-json"
)

// go-json does not implement the "omitzero" tag option yet, so Marshal routes
// types that use it to encoding/json. Flip this to true and the diversion, plus
// the type walk that feeds it, compile away.
const backendSupportsOmitZero = false

// go-json declares its own error types rather than reusing encoding/json's, and
// does not always classify a failure the same way, so errors are restated
// before they leave the package. Flip this to true and the replay compiles
// away.
const backendUsesStdErrors = false

func backendMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func backendUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// go-json builds its codecs lazily and exposes no pretouch hook, so preheating
// is a no-op here.

func backendPreheat(vt reflect.Type) error {
	return nil
}

func backendPreheatMany(vts []reflect.Type) error {
	return nil
}
