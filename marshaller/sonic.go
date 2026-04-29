//go:build (amd64 || arm64) && !illumos && !plan9 && !solaris

package marshaller

import (
	"reflect"

	"github.com/bytedance/sonic"
)

func Marshal(v any) ([]byte, error) {
	return sonic.Marshal(v)
}

func Unmarshal(data []byte, v any) error {
	return sonic.Unmarshal(data, v)
}

func Preheat(vt reflect.Type) error {
	return sonic.Pretouch(vt)
}

func PreheatMany(vts []reflect.Type) error {
	return sonic.PretouchMany(vts)
}
