package json

import (
	"go.gh.ink/json/marshaller"
)

func Marshal(v any) ([]byte, error) {
	return marshaller.Marshal(v)
}

func Unmarshal(data []byte, v any) error {
	return marshaller.Unmarshal(data, v)
}
