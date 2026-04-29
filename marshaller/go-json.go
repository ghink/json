//go:build !((amd64 || arm64) && !illumos && !plan9 && !solaris)

package marshaller

import (
	"reflect"

	"github.com/goccy/go-json"
)

func Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func Preheat(vt reflect.Type) error {
	return nil
}

func PreheatMany(vts []reflect.Type) error {
	return nil
}
