# json

A JSON library that delivers `encoding/json` semantics on a faster backend.

## Features

- `encoding/json` v1 semantics, held in place by a differential conformance
  suite and three fuzz targets.
- Keeps the `omitzero` tag option working even though the backend lacks it.
- Reports failures with `encoding/json`'s own error types.
- Contains a backend panic on malformed input rather than letting it out.

## What "same as encoding/json" means here

The backend is held to `encoding/json`'s **meaning**, not to its spelling.

A difference is worth paying for when it changes what the caller ends up with,
and those are routed to `encoding/json` instead. Most are worked out from the
type, so the cost lands once per type rather than once per call:

- **`omitzero`.** go-json does not implement the tag option, so a field that
  should have been dropped would be emitted.
- **Map key types.** Only string, integer and text-marshalling keys stay on the
  backend. A pointer key makes it write an unquoted member name, which is not
  valid JSON at all; float keys it rejects outright, while `encoding/json`
  accepts them from Go 1.27 on.
- **The `string` tag option, on decode.** The backend reads `"1.5"` into an int
  as `1` rather than refusing it. Encoding is unaffected and stays fast.
- **Invalid UTF-8 on decode.** `encoding/json` rewrites malformed bytes inside
  decoded strings to U+FFFD; the backend passes them through, leaving a
  different string in the target.
- **Errors.** The backend raises its own error types and does not always
  classify a failure the same way, so a failed call is replayed through
  `encoding/json` and its error returned instead. Only failures pay for it.

A difference that only changes how the same value is written is left alone,
because catching it meant scanning every result for no gain downstream. All of
these decode back to exactly what `encoding/json`'s output decodes to:

- Object members are ordered by the escaped form of their names rather than by
  the names, so `"&"` can land after `"B"` instead of before it.
- Floats keep the zero `strconv` pads a small negative exponent with: `1e-07`
  where `encoding/json` writes `1e-7`.
- Backspace and form feed are written as `\u0008` and `\u000c` rather than
  `\b` and `\f`, and a replaced U+FFFD is escaped rather than emitted
  literally.

If you compare JSON bytes across implementations, for snapshots, signatures or a
consumer in another language, that last group matters and this library is not
the right tool.

### Known leniency

The backend does not validate the parts of a document it skips, and reads some
number literals `encoding/json` rejects, so these are accepted where
`encoding/json` returns an error:

```
{"a":1,"junk":{{}}}   {"junk":[1,]}   {"junk":"\x41"}   {"0":00}   01   1.
```

Catching this means validating every document up front, which costs more than
the decode itself, so it is left as is. If you need malformed input rejected,
call `encoding/json.Valid` before `Unmarshal`. `TestDocumentedLeniency` pins the
behaviour so the note can go if the backend ever tightens up.

### Errors and partial state

Failures carry `encoding/json`'s error types, so the usual inspection works:

```go
var typeErr *stdjson.UnmarshalTypeError
if errors.As(err, &typeErr) {
    // typeErr.Field, typeErr.Offset, ...
}
```

When `Unmarshal` reports an error the contents of the target are unspecified:
the backend applies part of a malformed document before giving up. Treat a
failed decode as having produced nothing.

## Requirement

- Go: 1.25.0+

Also available: [`go.gh.ink/json/v2`](v2/), which delivers `encoding/json/v2`
semantics and requires Go 1.27. The two are separate modules and develop
independently.

## Usage

### Marshal/Unmarshal

```go
import "go.gh.ink/json"

var data YourSchema
// Marshal
output, err := json.Marshal(&data)
// Unmarshal
err := json.Unmarshal(output, &data)
```

### Preheat

Compiles codecs ahead of their first use. It is a no-op on backends with nothing
to compile.

```go
err := json.Preheat(reflect.TypeFor[YourSchema]())
err := json.PreheatMany([]reflect.Type{reflect.TypeFor[YourSchema]()})
```
