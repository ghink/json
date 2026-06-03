# json

A JSON library that uses conditional compilation to be compatible with different architectures.

Based on [bytedance/sonic](https://github.com/bytedance/sonic) and [goccy/go-json](https://github.com/goccy/go-json)

## Requirement

- Go: 1.25.0+

## Features

- Uses conditional compilation to be compatible with different architectures.

## Usage

### Marshal/Unmarshal

Default behaviors are mostly consistent with `encoding/json`.

 ```go
import "go.gh.ink/json"

var data YourSchema
// Marshal
output, err := json.Marshal(&data)
// Unmarshal
err := json.Unmarshal(output, &data)
 ```