package marshaller

import (
	jsonv2 "encoding/json/v2"
	"reflect"
)

// encoding/json/v2 is the reference implementation of the semantics this module
// promises, so the backend is a straight pass-through. Nothing here reshapes a
// call, and no option is filtered: whatever the caller asks for reaches v2
// unchanged.
//
// When a faster implementation of v2 semantics appears, it replaces this file.
// The bar it has to clear is byte-identical output against encoding/json/v2,
// the same way the v1 module holds its backend to encoding/json; anything it
// cannot honour is routed back here rather than allowed to differ.

func backendMarshal(v any, opts ...Options) ([]byte, error) {
	return jsonv2.Marshal(v, opts...)
}

func backendUnmarshal(data []byte, v any, opts ...Options) error {
	return jsonv2.Unmarshal(data, v, opts...)
}

// encoding/json/v2 builds its codecs lazily and exposes no pretouch hook, so
// preheating is a no-op here.

func backendPreheat(vt reflect.Type) error {
	return nil
}

func backendPreheatMany(vts []reflect.Type) error {
	return nil
}
