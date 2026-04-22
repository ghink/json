//go:build !((amd64 || arm64) && !illumos && !plan9 && !solaris)

package marshaller

import (
	"github.com/goccy/go-json"
)

func Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
