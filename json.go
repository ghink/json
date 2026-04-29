package json

import (
	"reflect"

	"go.gh.ink/json/marshaller"
)

func Marshal(v any) ([]byte, error) {
	return marshaller.Marshal(v)
}

func Unmarshal(data []byte, v any) error {
	return marshaller.Unmarshal(data, v)
}

func Preheat(vt reflect.Type) error {
	return marshaller.Preheat(vt)
}

func PreheatMany(vts []reflect.Type) error {
	return marshaller.PreheatMany(vts)
}
