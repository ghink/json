# json/v2

`encoding/json/v2` semantics behind an import path you control.

## Requirement

- Go: 1.27.0+

`encoding/json/v2` first ships in Go 1.27, and there is no way to provide its
semantics on an older toolchain without depending on the upstream prototype,
which declares itself subject to regular breaking changes. So this module
requires 1.27; the v1 module at `go.gh.ink/json` stays on 1.25 and keeps
developing independently.

Note that building with `GOEXPERIMENT=nojsonv2` removes `encoding/json/v2` and
`encoding/json/jsontext` from the standard library entirely, which this module
cannot work around.

## Why this exists

Today every call forwards straight to `encoding/json/v2`, so on performance this
module gives you nothing the standard library does not. What it gives you is the
seam.

When an implementation of v2 semantics appears that beats the standard library,
it slots in behind these functions. Callers pick it up by updating the
dependency; no business code changes, no repo-wide import rewrite. Write new
code against this path and that swap costs you a version bump.

The bar a backend has to clear is byte-identical output against
`encoding/json/v2`. [conformance_test.go](conformance_test.go) pins that, the
same way the v1 module pins its backend against `encoding/json`, so the swap can
be made without asking every caller to re-verify.

## Usage

 ```go
import "go.gh.ink/json/v2"

var data YourSchema
// Marshal
output, err := json.Marshal(&data)
// Unmarshal
err := json.Unmarshal(output, &data)
 ```

### Options

`Options` is `encoding/json/v2`'s own option type, so every constructor from
`encoding/json/v2`, `encoding/json/jsontext` and `encoding/json` applies:

 ```go
import (
    stdjson "encoding/json"
    "encoding/json/jsontext"
    jsonv2 "encoding/json/v2"

    "go.gh.ink/json/v2"
)

output, err := json.Marshal(&data,
    jsontext.WithIndent("  "),
    jsonv2.Deterministic(true),
)
 ```

Aliasing the type rather than defining one does not tie the module to the
standard library: `jsonv2.GetOption` reads an option back out of a value and
reports whether it was set at all, so a future backend can be handed the ones it
understands while the rest fall through to `encoding/json/v2`.

### Migrating from v1

v2 changes observable behaviour. Among others, it does not escape HTML
characters by default, encodes a nil slice as `[]` rather than `null`, and
matches object member names case-sensitively. Anything asserting on your JSON
bytes, such as snapshot tests, cross-language consumers or signatures, has to be
checked.

If some consumers are still pinned to v1 output, run this module with v1
semantics on the v2 engine until they catch up:

 ```go
output, err := json.Marshal(&data, stdjson.DefaultOptionsV1())
 ```

That combination is byte-identical to `encoding/json`, which
[conformance_test.go](conformance_test.go) checks over the whole case table.

### Preheat

A no-op today, kept so that callers written against this package keep working
when a backend that precompiles arrives.

 ```go
err := json.Preheat(reflect.TypeFor[YourSchema]())
err := json.PreheatMany([]reflect.Type{reflect.TypeFor[YourSchema]()})
 ```
