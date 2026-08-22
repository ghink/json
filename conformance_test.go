package json_test

import (
	stdjson "encoding/json"
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	"go.gh.ink/json"
)

// The library promises encoding/json v1 semantics on a faster backend. These
// tests hold it to that by running every case through both and comparing.

type basic struct {
	A int    `json:"a"`
	B string `json:"b"`
	C bool   `json:"c"`
}

type omitted struct {
	Empty  string  `json:"empty,omitempty"`
	Zero   int     `json:"zero,omitzero"`
	Both   float64 `json:"both,omitempty,omitzero"`
	Skip   string  `json:"-"`
	Dash   string  `json:"-,"`
	Plain  string  `json:"plain"`
	NoName int     `json:",omitempty"`
}

type stringOpt struct {
	I int     `json:"i,string"`
	F float64 `json:"f,string"`
	B bool    `json:"b,string"`
	S string  `json:"s,string"`
}

type inner struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type embedsValue struct {
	inner
	Z int `json:"z"`
}

type embedsPointer struct {
	*inner
	Z int `json:"z"`
}

type embedsNamed struct {
	Inner inner `json:"nested"`
}

// Both levels declare "x"; the shallower one wins in encoding/json.
type shadowing struct {
	inner
	X string `json:"x"`
}

type valueMarshaler struct{ N int }

func (v valueMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(`{"vm":` + stdjson.Number(itoa(v.N)).String() + `}`), nil
}

type pointerMarshaler struct{ N int }

func (p *pointerMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(`{"pm":` + itoa(p.N) + `}`), nil
}

type textMarshaler struct{ S string }

func (t textMarshaler) MarshalText() ([]byte, error) { return []byte("T:" + t.S), nil }

type errMarshaler struct{}

func (errMarshaler) MarshalJSON() ([]byte, error) { return nil, errors.New("boom") }

type cyclic struct {
	Name string  `json:"name"`
	Next *cyclic `json:"next"`
}

type withInterface struct {
	V any `json:"v"`
}

type withRaw struct {
	R stdjson.RawMessage `json:"r"`
}

type withNumber struct {
	N stdjson.Number `json:"n"`
}

type withTime struct {
	T time.Time     `json:"t"`
	D time.Duration `json:"d"`
}

type withBytes struct {
	B  []byte   `json:"b"`
	A4 [4]byte  `json:"a4"`
	BB [][]byte `json:"bb"`
	NB []byte   `json:"nb"`
	PB *[]byte  `json:"pb"`
}

type unexportedHolder struct {
	Exported   int `json:"exported"`
	unexported int
	_          int
}

type namedString string

type namedInt int

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func marshalCases() map[string]any {
	pb := []byte("ptr")
	return map[string]any{
		// scalars
		"nil":            nil,
		"bool":           true,
		"int":            42,
		"int-min":        math.MinInt64,
		"int-max":        math.MaxInt64,
		"uint-max":       uint64(math.MaxUint64),
		"float":          1.5,
		"float-neg-zero": math.Copysign(0, -1),
		"float-max":      math.MaxFloat64,
		"float-small":    5e-324,
		"float-1e21":     1e21,
		"float-1e-7":     1e-7,
		"float-int-like": float64(1),

		// error-producing floats
		"nan":     math.NaN(),
		"inf":     math.Inf(1),
		"neg-inf": math.Inf(-1),

		// strings
		"string":           "hello",
		"string-html":      "<b>&\"quoted\"</b>",
		"string-unicode":   "héllo 世界 🎉",
		"string-ctrl":      "a\x00b\x1fc\td\ne",
		"string-lineterm":  "a\u2028b\u2029c",
		"string-badutf8":   string([]byte{0x61, 0xff, 0xfe, 0x62}),
		"string-solidus":   "a/b",
		"string-backslash": `a\b`,

		// nil-ish
		"nil-slice":          []int(nil),
		"nil-map":            map[string]int(nil),
		"nil-ptr":            (*basic)(nil),
		"nil-iface-fld":      withInterface{},
		"typed-nil-in-iface": withInterface{V: (*basic)(nil)},
		"ptr-to-ptr":         func() **int { i := 7; p := &i; return &p }(),

		// collections
		"empty-slice":  []int{},
		"slice":        []int{1, 2, 3},
		"nested-slice": [][]int{{1}, nil, {}},
		"array":        [3]int{1, 2, 3},
		"empty-map":    map[string]int{},
		"map-sorted":   map[string]int{"z": 1, "a": 2, "M": 3, "0": 4},
		"map-int-key":  map[int]string{10: "a", 2: "b", -1: "c"},
		"map-text-key": map[textMarshaler]int{{S: "b"}: 1, {S: "a"}: 2},
		"map-any":      map[string]any{"n": nil, "s": "x", "f": 1.0, "b": false},

		// structs
		"struct":         basic{A: 1, B: "two", C: true},
		"struct-ptr":     &basic{A: 1},
		"omitted":        omitted{},
		"omitted-filled": omitted{Empty: "e", Zero: 1, Both: 2, Skip: "s", Dash: "d", Plain: "p", NoName: 3},
		"string-opt":     stringOpt{I: 1, F: 2.5, B: true, S: "q"},
		"embeds-value":   embedsValue{inner: inner{X: 1, Y: 2}, Z: 3},
		"embeds-pointer": embedsPointer{inner: &inner{X: 1}, Z: 3},
		"embeds-nil-ptr": embedsPointer{Z: 3},
		"embeds-named":   embedsNamed{Inner: inner{X: 1}},
		"shadowing":      shadowing{inner: inner{X: 1, Y: 2}, X: "win"},
		"unexported":     unexportedHolder{Exported: 1, unexported: 2},

		// marshaler protocols
		"value-marshaler":     valueMarshaler{N: 5},
		"value-marshaler-ptr": &valueMarshaler{N: 5},
		"pointer-marshaler":   &pointerMarshaler{N: 6},
		"marshaler-in-slice":  []valueMarshaler{{N: 1}, {N: 2}},
		"marshaler-in-map":    map[string]valueMarshaler{"k": {N: 1}},
		"text-marshaler":      textMarshaler{S: "hi"},
		"err-marshaler":       errMarshaler{},
		"err-marshaler-nested": struct {
			E errMarshaler `json:"e"`
		}{},

		// std types
		"raw":         withRaw{R: stdjson.RawMessage(`{"k":[1,2]}`)},
		"raw-nil":     withRaw{},
		"number":      withNumber{N: "123.45"},
		"number-bare": stdjson.Number("1e3"),
		"time":        withTime{T: time.Unix(1700000000, 123456789).UTC(), D: 90 * time.Second},
		"bytes":       withBytes{B: []byte("hi"), A4: [4]byte{1, 2, 3, 4}, BB: [][]byte{[]byte("a")}, PB: &pb},

		// failures
		"chan":         make(chan int),
		"func":         func() {},
		"chan-in-strc": struct{ C chan int }{},
		"cyclic":       func() *cyclic { c := &cyclic{Name: "a"}; c.Next = c; return c }(),

		// map key types, whose handling differs between the backend and
		// encoding/json, and between Go releases
		"map-key-named-string": map[namedString]int{"a": 1, "b": 2},
		"map-key-int8":         map[int8]int{3: 1, 1: 2},
		"map-key-uint64":       map[uint64]int{3: 1, 1: 2},
		"map-key-named-int":    map[namedInt]string{2: "b", 1: "a"},
		"map-key-float64":      map[float64]int{1.5: 1, 2.0: 2},
		"map-key-float32":      map[float32]int{1.5: 1},
		"map-key-bool":         map[bool]int{true: 1},
		"map-key-ptr":          func() any { i := 0; return map[*int]int{&i: 1} }(),
		"map-key-array":        map[[2]int]int{{1, 2}: 1},
		"map-key-struct":       map[inner]int{{X: 1}: 1},
		"map-key-iface":        map[any]int{"a": 1},
		"map-key-complex":      map[complex128]int{1: 1},
		"map-key-nested":       map[string]map[float64]int{"k": {1.5: 1}},
		"map-key-in-slice":     []map[bool]int{{true: 1}},

		// invalid UTF-8, which the backend escapes and encoding/json stopped
		// escaping in Go 1.27
		"badutf8-single":  string([]byte{0xff}),
		"badutf8-in-strc": basic{B: string([]byte{0x61, 0xff, 0x62})},
		"badutf8-in-map":  map[string]string{"k": string([]byte{0xff})},
		"badutf8-in-key":  map[string]int{string([]byte{0xff}): 1},
		"real-fffd":       "a�b",
		"literal-escape":  `a�b`,
	}
}

func TestMarshalConformance(t *testing.T) {
	for name, v := range marshalCases() {
		t.Run(name, func(t *testing.T) {
			want, wantErr := stdjson.Marshal(v)
			got, gotErr := json.Marshal(v)

			if (wantErr != nil) != (gotErr != nil) {
				t.Fatalf("error mismatch:\n  encoding/json: %v\n  go.gh.ink/json: %v", wantErr, gotErr)
			}
			if wantErr != nil {
				return
			}
			if string(got) != string(want) && !sameJSON(t, got, want) {
				t.Errorf("output mismatch:\n  encoding/json: %s\n  go.gh.ink/json: %s", want, got)
			}
		})
	}
}

// unmarshalCase pairs an input document with a factory producing a fresh zero
// value of the target type, so both libraries decode into identical state.
type unmarshalCase struct {
	data string
	newV func() any
}

func unmarshalCases() map[string]unmarshalCase {
	return map[string]unmarshalCase{
		"struct":            {`{"a":1,"b":"x","c":true}`, func() any { return new(basic) }},
		"struct-case-fold":  {`{"A":1,"B":"x","C":true}`, func() any { return new(basic) }},
		"struct-unknown":    {`{"a":1,"zzz":9}`, func() any { return new(basic) }},
		"struct-dup-keys":   {`{"a":1,"a":2}`, func() any { return new(basic) }},
		"struct-null-field": {`{"a":null,"b":null}`, func() any { return new(basic) }},
		"struct-partial":    {`{"b":"only"}`, func() any { return new(basic) }},
		"struct-wrong-type": {`{"a":"nope"}`, func() any { return new(basic) }},
		"struct-into-num":   {`5`, func() any { return new(basic) }},

		"embeds-value":   {`{"x":1,"y":2,"z":3}`, func() any { return new(embedsValue) }},
		"embeds-pointer": {`{"x":1,"z":3}`, func() any { return new(embedsPointer) }},
		"shadowing":      {`{"x":"s","y":2}`, func() any { return new(shadowing) }},
		"named-nested":   {`{"nested":{"x":1}}`, func() any { return new(embedsNamed) }},

		"omitted":    {`{"plain":"p","-":"dash","NoName":4}`, func() any { return new(omitted) }},
		"string-opt": {`{"i":"1","f":"2.5","b":"true","s":"\"q\""}`, func() any { return new(stringOpt) }},

		"slice":       {`[1,2,3]`, func() any { return new([]int) }},
		"slice-null":  {`null`, func() any { return new([]int) }},
		"slice-empty": {`[]`, func() any { return new([]int) }},
		"array-short": {`[1]`, func() any { return new([3]int) }},
		"array-long":  {`[1,2,3,4,5]`, func() any { return new([3]int) }},

		"map":        {`{"b":2,"a":1}`, func() any { return new(map[string]int) }},
		"map-int":    {`{"1":10,"2":20}`, func() any { return new(map[int]string) }},
		"map-any":    {`{"n":null,"s":"x","f":1,"b":false,"arr":[1,{"k":2}]}`, func() any { return new(map[string]any) }},
		"into-any":   {`{"a":[1,2,{"b":null}],"c":1e3}`, func() any { return new(any) }},
		"any-bignum": {`12345678901234567890123`, func() any { return new(any) }},

		"bytes":        {`{"b":"aGk=","a4":[1,2,3,4],"bb":["YQ=="],"nb":null}`, func() any { return new(withBytes) }},
		"bytes-badb64": {`{"b":"!!!"}`, func() any { return new(withBytes) }},
		"raw":          {`{"r":{"k":[1,2]}}`, func() any { return new(withRaw) }},
		"number":       {`{"n":123.45}`, func() any { return new(withNumber) }},
		"number-bad":   {`{"n":"abc"}`, func() any { return new(withNumber) }},
		"time":         {`{"t":"2023-11-14T22:13:20.123456789Z","d":90000000000}`, func() any { return new(withTime) }},
		"time-bad":     {`{"t":"not-a-time"}`, func() any { return new(withTime) }},

		"int-overflow":   {`{"a":99999999999999999999}`, func() any { return new(basic) }},
		"float-into-int": {`{"a":1.5}`, func() any { return new(basic) }},
		"exp-into-int":   {`{"a":1e2}`, func() any { return new(basic) }},

		"escapes":        {`{"b":"a\u0041b\/c\\d\"e\n"}`, func() any { return new(basic) }},
		"surrogate":      {`{"b":"\ud83c\udf89"}`, func() any { return new(basic) }},
		"lone-surrogate": {`{"b":"\ud800"}`, func() any { return new(basic) }},
		"badutf8-input":  {"{\"b\":\"a\xffb\"}", func() any { return new(basic) }},

		"syntax-trailing":  {`{"a":1},`, func() any { return new(basic) }},
		"syntax-truncated": {`{"a":`, func() any { return new(basic) }},
		"syntax-empty":     {``, func() any { return new(basic) }},
		"syntax-garbage":   {`@!#`, func() any { return new(basic) }},
		"whitespace":       {"  \t\n{\"a\":1}\r\n ", func() any { return new(basic) }},
		"deep-nest":        {`{"v":[[[[[1]]]]]}`, func() any { return new(withInterface) }},

		"map-key-float":   {`{"1.5":10}`, func() any { return new(map[float64]int) }},
		"map-key-bool":    {`{"true":10}`, func() any { return new(map[bool]int) }},
		"map-key-int8":    {`{"3":"a"}`, func() any { return new(map[int8]string) }},
		"map-key-named":   {`{"a":1}`, func() any { return new(map[namedString]int) }},
		"map-key-nested":  {`{"k":{"1.5":10}}`, func() any { return new(map[string]map[float64]int) }},
		"badutf8-single":  {"\"\xff\"", func() any { return new(string) }},
		"badutf8-in-key":  {"{\"\xff\":1}", func() any { return new(map[string]int) }},
		"badutf8-in-any":  {"{\"b\":\"a\xffb\"}", func() any { return new(map[string]any) }},
		"float-overflow":  {`1e400`, func() any { return new(float64) }},
		"neg-zero":        {`-0`, func() any { return new(float64) }},
		"precise-float":   {`1.0000000000000000000000001`, func() any { return new(float64) }},
		"base64-unpadded": {`{"b":"aGk"}`, func() any { return new(withBytes) }},
		"string-opt-bare": {`{"i":1}`, func() any { return new(stringOpt) }},
	}
}

// raceUnsafeCases are inputs the backend decodes with pointer arithmetic that
// steps outside its buffer. Ordinarily the read lands on whatever follows and
// goes unnoticed; under the race detector checkptr turns it into an
// unrecoverable "fatal error: checkptr", which no amount of recovering in this
// package can contain. They are skipped there so the rest of the suite, and the
// type cache in particular, still gets race coverage.
//
// Reported upstream as goccy/go-json#575. Drop entries as that is fixed.
var raceUnsafeCases = map[string]string{
	"lone-surrogate": "unescapeString reads past the buffer on a lone surrogate escape (internal/decoder/string.go)",
}

func TestUnmarshalConformance(t *testing.T) {
	for name, c := range unmarshalCases() {
		t.Run(name, func(t *testing.T) {
			if reason, unsafe := raceUnsafeCases[name]; unsafe && raceDetectorEnabled {
				t.Skipf("skipped under -race: %s", reason)
			}
			wantV, gotV := c.newV(), c.newV()
			wantErr := stdjson.Unmarshal([]byte(c.data), wantV)
			gotErr := json.Unmarshal([]byte(c.data), gotV)

			if (wantErr != nil) != (gotErr != nil) {
				t.Fatalf("error mismatch:\n  encoding/json: %v\n  go.gh.ink/json: %v", wantErr, gotErr)
			}
			if wantErr != nil {
				// Both rejected the document. The library documents the target
				// as unspecified past an error, so only the message is held to
				// encoding/json's wording.
				if gotErr.Error() != wantErr.Error() {
					t.Errorf("error message mismatch on %q:\n  encoding/json: %v\n  go.gh.ink/json: %v", c.data, wantErr, gotErr)
				}
				return
			}
			if !reflect.DeepEqual(wantV, gotV) {
				t.Errorf("decoded state mismatch on %q:\n  encoding/json: %#v\n  go.gh.ink/json: %#v", c.data, wantV, gotV)
			}
		})
	}
}

// TestErrorMessages holds the restated errors to encoding/json's exact wording,
// not merely its types.
func TestErrorMessages(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		for name, v := range marshalCases() {
			_, wantErr := stdjson.Marshal(v)
			if wantErr == nil {
				continue
			}
			_, gotErr := json.Marshal(v)
			if gotErr == nil {
				t.Errorf("%s: got nil, want %v", name, wantErr)
				continue
			}
			if gotErr.Error() != wantErr.Error() {
				t.Errorf("%s:\n  encoding/json: %v\n  go.gh.ink/json: %v", name, wantErr, gotErr)
			}
		}
	})
}

// TestErrorTypes covers downstream code that type-asserts on the errors these
// calls return; a different concrete type breaks it even when the message reads
// the same.
func TestErrorTypes(t *testing.T) {
	t.Run("UnmarshalTypeError", func(t *testing.T) {
		var v basic
		err := json.Unmarshal([]byte(`{"a":"str"}`), &v)
		var target *stdjson.UnmarshalTypeError
		if !errors.As(err, &target) {
			t.Errorf("got %T (%v), want *encoding/json.UnmarshalTypeError", err, err)
		}
	})

	t.Run("SyntaxError", func(t *testing.T) {
		var v basic
		err := json.Unmarshal([]byte(`{"a":`), &v)
		var target *stdjson.SyntaxError
		if !errors.As(err, &target) {
			t.Errorf("got %T (%v), want *encoding/json.SyntaxError", err, err)
		}
	})

	t.Run("InvalidUnmarshalError", func(t *testing.T) {
		var v basic
		err := json.Unmarshal([]byte(`{}`), v) // not a pointer
		var target *stdjson.InvalidUnmarshalError
		if !errors.As(err, &target) {
			t.Errorf("got %T (%v), want *encoding/json.InvalidUnmarshalError", err, err)
		}
	})

	t.Run("UnsupportedTypeError", func(t *testing.T) {
		_, err := json.Marshal(make(chan int))
		var target *stdjson.UnsupportedTypeError
		if !errors.As(err, &target) {
			t.Errorf("got %T (%v), want *encoding/json.UnsupportedTypeError", err, err)
		}
	})

	t.Run("UnsupportedValueError", func(t *testing.T) {
		_, err := json.Marshal(math.NaN())
		var target *stdjson.UnsupportedValueError
		if !errors.As(err, &target) {
			t.Errorf("got %T (%v), want *encoding/json.UnsupportedValueError", err, err)
		}
	})

	t.Run("MarshalerError", func(t *testing.T) {
		_, err := json.Marshal(errMarshaler{})
		var target *stdjson.MarshalerError
		if !errors.As(err, &target) {
			t.Errorf("got %T (%v), want *encoding/json.MarshalerError", err, err)
		}
	})
}

// TestRoundTrip checks that a value survives encode/decode through this library
// and lands where encoding/json would land.
func TestRoundTrip(t *testing.T) {
	values := []any{
		basic{A: 1, B: "x", C: true},
		omitted{Plain: "p", Zero: 3},
		embedsValue{inner: inner{X: 1, Y: 2}, Z: 3},
		stringOpt{I: 1, F: 2.5, B: true, S: "q"},
		withBytes{B: []byte("hi"), A4: [4]byte{1, 2, 3, 4}},
		map[string]any{"a": float64(1), "b": []any{float64(2)}},
	}
	for _, v := range values {
		blob, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("Marshal(%T): %v", v, err)
		}
		out := reflect.New(reflect.TypeOf(v))
		if err := json.Unmarshal(blob, out.Interface()); err != nil {
			t.Fatalf("Unmarshal(%T): %v", v, err)
		}
		std := reflect.New(reflect.TypeOf(v))
		if err := stdjson.Unmarshal(blob, std.Interface()); err != nil {
			t.Fatalf("std Unmarshal(%T): %v", v, err)
		}
		if !reflect.DeepEqual(out.Interface(), std.Interface()) {
			t.Errorf("round trip diverged for %T:\n  encoding/json: %#v\n  go.gh.ink/json: %#v",
				v, std.Interface(), out.Interface())
		}
	}
}

// TestDocumentedLeniency pins the one place this library knowingly parts ways
// with encoding/json: the backend does not validate what it skips, and reads
// number literals encoding/json rejects. Catching this would mean validating
// every document up front, which costs more than the decode itself, so it is
// documented in README.md instead.
//
// The cases are asserted to keep being accepted. If the backend tightens up,
// this test fails and both it and the README note can go.
func TestDocumentedLeniency(t *testing.T) {
	cases := []struct {
		data string
		newV func() any
	}{
		{`{"a":1,"junk":{{}}}`, func() any { return new(basic) }},
		{`{"junk":[1,]}`, func() any { return new(basic) }},
		{`{"junk":"\x41"}`, func() any { return new(basic) }},
		{`{"junk":{"nested":}}`, func() any { return new(basic) }},
		{`{"junk":01}`, func() any { return new(basic) }},
		{`{"0":00}`, func() any { return new(any) }},
		{`01`, func() any { return new(any) }},
		{`1.`, func() any { return new(any) }},
	}

	for _, c := range cases {
		t.Run(c.data, func(t *testing.T) {
			if stdjson.Valid([]byte(c.data)) {
				t.Fatalf("%q is well formed; it does not belong in this table", c.data)
			}
			if err := stdjson.Unmarshal([]byte(c.data), c.newV()); err == nil {
				t.Fatalf("encoding/json accepted %q; it does not belong in this table", c.data)
			}
			if err := json.Unmarshal([]byte(c.data), c.newV()); err != nil {
				t.Errorf("%q is now rejected (%v); the backend got stricter, so drop this case and the README note", c.data, err)
			}
		})
	}
}

// sameJSON reports whether two documents carry the same value. The backend is
// held to encoding/json's meaning rather than its spelling, so a result that
// differs only in member order, exponent padding or the long form of an escape
// is accepted; anything that decodes to a different value is not.
func sameJSON(t *testing.T, got, want []byte) bool {
	t.Helper()
	var gotV, wantV any
	if err := stdjson.Unmarshal(got, &gotV); err != nil {
		t.Errorf("this library produced undecodable output %s: %v", got, err)
		return false
	}
	if err := stdjson.Unmarshal(want, &wantV); err != nil {
		t.Fatalf("encoding/json produced undecodable output %s: %v", want, err)
	}
	return reflect.DeepEqual(gotV, wantV)
}

// TestNoPanicOnTruncatedInput covers the guard around the backend. goccy/go-json
// indexes past the end of its buffer decoding an object member name, so a
// truncated escape can panic instead of returning an error; Unmarshal contains
// that and re-runs the document through encoding/json.
//
// Whether any single input trips the read depends on what happens to sit past
// the buffer, so this walks a range of truncations rather than pinning one.
func TestNoPanicOnTruncatedInput(t *testing.T) {
	full := `{"a":1,"b":"vaAlue","c":true,"nested":{"x":[1,2,{"y":"é"}]}}`
	targets := []func() any{
		func() any { return new(basic) },
		func() any { return new(omitted) },
		func() any { return new(embedsValue) },
		func() any { return new(withInterface) },
		func() any { return new(map[string]any) },
		func() any { return new(any) },
	}

	for cut := range len(full) {
		data := []byte(full[:cut])
		for i, newV := range targets {
			wantErr := stdjson.Unmarshal(data, newV())

			// A panic escaping here fails the test by crashing it, which is the
			// point; the assertions below cover the quieter half.
			gotErr := json.Unmarshal(data, newV())

			if (wantErr != nil) != (gotErr != nil) {
				t.Errorf("target %d, %q: error mismatch:\n  encoding/json: %v\n  go.gh.ink/json: %v",
					i, data, wantErr, gotErr)
			}
		}
	}
}
