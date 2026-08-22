package json_test

import (
	stdjson "encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	"go.gh.ink/json/v2"
)

// This module promises encoding/json/v2 semantics. The tests below pin that
// promise, so the day a different backend slots in behind these calls the suite
// says whether it actually matches.

type basic struct {
	A int    `json:"a"`
	B string `json:"b"`
	C bool   `json:"c"`
}

type omitted struct {
	Empty string  `json:"empty,omitempty"`
	Zero  int     `json:"zero,omitzero"`
	Both  float64 `json:"both,omitempty,omitzero"`
	Skip  string  `json:"-"`
	Plain string  `json:"plain"`
}

type inner struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type embeds struct {
	inner
	Z int `json:"z"`
}

type nilly struct {
	S []int          `json:"s"`
	M map[string]int `json:"m"`
	P *inner         `json:"p"`
}

type marshaler struct{ N int }

func (marshaler) MarshalJSON() ([]byte, error) { return []byte(`{"m":1}`), nil }

type texty struct{ S string }

func (t texty) MarshalText() ([]byte, error) { return []byte("T:" + t.S), nil }

type withTime struct {
	T time.Time     `json:"t"`
	D time.Duration `json:"d"`
}

type withBytes struct {
	B  []byte  `json:"b"`
	A4 [4]byte `json:"a4"`
}

func marshalCases() map[string]any {
	return map[string]any{
		"nil":            nil,
		"bool":           true,
		"int-max":        math.MaxInt64,
		"uint-max":       uint64(math.MaxUint64),
		"float":          1.5,
		"float-1e-7":     1e-7,
		"float-1e21":     1e21,
		"float-neg-zero": math.Copysign(0, -1),
		"nan":            math.NaN(),
		"inf":            math.Inf(1),

		"string-html":    "<b>&\"q\"</b>",
		"string-unicode": "héllo 世界 🎉",
		"string-ctrl":    "a\x00b\tc\nd",
		"string-badutf8": string([]byte{0x61, 0xff, 0x62}),

		"nil-slice": []int(nil),
		"nil-map":   map[string]int(nil),
		"nil-ptr":   (*basic)(nil),
		"nilly":     nilly{},

		"empty-slice": []int{},
		"slice":       []int{1, 2, 3},
		"array":       [3]int{1, 2, 3},
		"map":         map[string]int{"z": 1, "a": 2, "M": 3},
		"map-int-key": map[int]string{10: "a", 2: "b"},
		"map-any":     map[string]any{"n": nil, "s": "x"},

		"struct":         basic{A: 1, B: "two", C: true},
		"struct-ptr":     &basic{A: 1},
		"omitted":        omitted{},
		"omitted-filled": omitted{Empty: "e", Zero: 1, Both: 2, Skip: "s", Plain: "p"},
		"embeds":         embeds{inner: inner{X: 1, Y: 2}, Z: 3},

		"marshaler": marshaler{N: 5},
		"texty":     texty{S: "hi"},

		"raw":    stdjson.RawMessage(`{"k":[1,2]}`),
		"number": stdjson.Number("1e3"),
		"time":   withTime{T: time.Unix(1700000000, 123456789).UTC(), D: 90 * time.Second},
		"bytes":  withBytes{B: []byte("hi"), A4: [4]byte{1, 2, 3, 4}},

		"chan":   make(chan int),
		"func":   func() {},
		"cyclic": func() any { m := map[string]any{}; m["self"] = m; return m }(),
	}
}

func TestMarshalMatchesStdV2(t *testing.T) {
	// Unlike v1, encoding/json/v2 leaves map member order unspecified unless
	// asked, so two independent calls need Deterministic before their bytes
	// can be compared at all. TestOptionsReachTheBackend covers the default.
	const deterministic = true

	for name, v := range marshalCases() {
		t.Run(name, func(t *testing.T) {
			want, wantErr := jsonv2.Marshal(v, jsonv2.Deterministic(deterministic))
			got, gotErr := json.Marshal(v, jsonv2.Deterministic(deterministic))

			if (wantErr != nil) != (gotErr != nil) {
				t.Fatalf("error mismatch:\n  encoding/json/v2: %v\n  go.gh.ink/json/v2: %v", wantErr, gotErr)
			}
			if wantErr != nil {
				if gotErr.Error() != wantErr.Error() {
					t.Errorf("error message mismatch:\n  encoding/json/v2: %v\n  go.gh.ink/json/v2: %v", wantErr, gotErr)
				}
				return
			}
			if string(got) != string(want) {
				t.Errorf("output mismatch:\n  encoding/json/v2: %s\n  go.gh.ink/json/v2: %s", want, got)
			}
		})
	}
}

type unmarshalCase struct {
	data string
	newV func() any
}

func unmarshalCases() map[string]unmarshalCase {
	return map[string]unmarshalCase{
		"struct":           {`{"a":1,"b":"x","c":true}`, func() any { return new(basic) }},
		"struct-case-fold": {`{"A":1}`, func() any { return new(basic) }},
		"struct-unknown":   {`{"a":1,"zzz":9}`, func() any { return new(basic) }},
		"struct-dup-keys":  {`{"a":1,"a":2}`, func() any { return new(basic) }},
		"struct-null":      {`{"a":null,"b":null}`, func() any { return new(basic) }},
		"struct-bad-type":  {`{"a":"nope"}`, func() any { return new(basic) }},
		"embeds":           {`{"x":1,"y":2,"z":3}`, func() any { return new(embeds) }},
		"omitted":          {`{"plain":"p","zero":3}`, func() any { return new(omitted) }},

		"slice":       {`[1,2,3]`, func() any { return new([]int) }},
		"slice-null":  {`null`, func() any { return new([]int) }},
		"array-short": {`[1]`, func() any { return new([3]int) }},
		"array-long":  {`[1,2,3,4,5]`, func() any { return new([3]int) }},

		"map":      {`{"b":2,"a":1}`, func() any { return new(map[string]int) }},
		"map-int":  {`{"1":10,"2":20}`, func() any { return new(map[int]string) }},
		"into-any": {`{"a":[1,2,{"b":null}],"c":1e3}`, func() any { return new(any) }},

		"bytes":  {`{"b":"aGk=","a4":[1,2,3,4]}`, func() any { return new(withBytes) }},
		"time":   {`{"t":"2023-11-14T22:13:20.123456789Z","d":"1m30s"}`, func() any { return new(withTime) }},
		"number": {`{"a":1}`, func() any { return new(basic) }},

		"int-overflow": {`{"a":99999999999999999999}`, func() any { return new(basic) }},
		"float-to-int": {`{"a":1.5}`, func() any { return new(basic) }},
		"escapes":      {`{"b":"a\u0041b\/c\\d\"e\n"}`, func() any { return new(basic) }},
		"surrogate":    {`{"b":"\ud83c\udf89"}`, func() any { return new(basic) }},

		"syntax-trailing":  {`{"a":1},`, func() any { return new(basic) }},
		"syntax-truncated": {`{"a":`, func() any { return new(basic) }},
		"syntax-empty":     {``, func() any { return new(basic) }},
		"whitespace":       {"  \t\n{\"a\":1}\r\n ", func() any { return new(basic) }},
		"not-a-pointer":    {`{}`, func() any { return basic{} }},
	}
}

func TestUnmarshalMatchesStdV2(t *testing.T) {
	for name, c := range unmarshalCases() {
		t.Run(name, func(t *testing.T) {
			wantV, gotV := c.newV(), c.newV()
			wantErr := jsonv2.Unmarshal([]byte(c.data), wantV)
			gotErr := json.Unmarshal([]byte(c.data), gotV)

			if (wantErr != nil) != (gotErr != nil) {
				t.Fatalf("error mismatch:\n  encoding/json/v2: %v\n  go.gh.ink/json/v2: %v", wantErr, gotErr)
			}
			if wantErr != nil && gotErr.Error() != wantErr.Error() {
				t.Errorf("error message mismatch on %q:\n  encoding/json/v2: %v\n  go.gh.ink/json/v2: %v", c.data, wantErr, gotErr)
			}
			if !reflect.DeepEqual(wantV, gotV) {
				t.Errorf("decoded state mismatch on %q:\n  encoding/json/v2: %#v\n  go.gh.ink/json/v2: %#v", c.data, wantV, gotV)
			}
		})
	}
}

// TestOptionsReachTheBackend guards the seam itself: an option the caller passes
// has to survive the hop through this module. A backend that quietly dropped
// one would still look correct on the tests above, which pass no options.
func TestOptionsReachTheBackend(t *testing.T) {
	type payload struct {
		Nil  []int  `json:"nil"`
		HTML string `json:"html"`
	}
	v := payload{HTML: "<a>&b"}

	cases := []struct {
		name string
		opts []json.Options
	}{
		{"none", nil},
		{"escape-html", []json.Options{jsontext.EscapeForHTML(true)}},
		{"nil-slice-as-null", []json.Options{jsonv2.FormatNilSliceAsNull(true)}},
		{"indent", []json.Options{jsontext.WithIndent("  ")}},
		{"deterministic", []json.Options{jsonv2.Deterministic(true)}},
		{"joined", []json.Options{jsonv2.JoinOptions(
			jsontext.EscapeForHTML(true),
			jsonv2.FormatNilSliceAsNull(true),
		)}},
		{"v1-preset", []json.Options{stdjson.DefaultOptionsV1()}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			want, wantErr := jsonv2.Marshal(v, c.opts...)
			got, gotErr := json.Marshal(v, c.opts...)
			if wantErr != nil || gotErr != nil {
				t.Fatalf("unexpected errors: %v / %v", wantErr, gotErr)
			}
			if string(got) != string(want) {
				t.Errorf("option dropped:\n  encoding/json/v2: %s\n  go.gh.ink/json/v2: %s", want, got)
			}
		})
	}

	// Decoding takes options too.
	for _, strict := range []bool{false, true} {
		var wantV, gotV basic
		wantErr := jsonv2.Unmarshal([]byte(`{"a":1,"nope":2}`), &wantV, jsonv2.RejectUnknownMembers(strict))
		gotErr := json.Unmarshal([]byte(`{"a":1,"nope":2}`), &gotV, jsonv2.RejectUnknownMembers(strict))
		if (wantErr != nil) != (gotErr != nil) || wantV != gotV {
			t.Errorf("RejectUnknownMembers(%v) diverged: %v/%+v vs %v/%+v", strict, wantErr, wantV, gotErr, gotV)
		}
	}
}

// TestV1PresetMatchesV1 pins the migration path documented on Options: running
// v2 with DefaultOptionsV1 has to reproduce encoding/json byte for byte, so a
// caller with consumers still pinned to v1 output can move over.
func TestV1PresetMatchesV1(t *testing.T) {
	for name, v := range marshalCases() {
		t.Run(name, func(t *testing.T) {
			want, wantErr := stdjson.Marshal(v)
			got, gotErr := json.Marshal(v, stdjson.DefaultOptionsV1())
			if (wantErr != nil) != (gotErr != nil) {
				t.Fatalf("error mismatch:\n  encoding/json: %v\n  go.gh.ink/json/v2: %v", wantErr, gotErr)
			}
			if wantErr != nil {
				return
			}
			if string(got) != string(want) {
				t.Errorf("output mismatch:\n  encoding/json: %s\n  go.gh.ink/json/v2: %s", want, got)
			}
		})
	}
}

// TestErrorTypes checks that v2's error types survive the hop, since downstream
// inspects them.
func TestErrorTypes(t *testing.T) {
	var v basic
	err := json.Unmarshal([]byte(`{"a":"str"}`), &v)
	var semantic *jsonv2.SemanticError
	if !errors.As(err, &semantic) {
		t.Errorf("got %T (%v), want *encoding/json/v2.SemanticError", err, err)
	}

	err = json.Unmarshal([]byte(`{"a":`), &v)
	var syntactic *jsontext.SyntacticError
	if !errors.As(err, &syntactic) {
		t.Errorf("got %T (%v), want *encoding/json/jsontext.SyntacticError", err, err)
	}
}

func TestRoundTrip(t *testing.T) {
	values := []any{
		basic{A: 1, B: "x", C: true},
		omitted{Plain: "p", Zero: 3},
		embeds{inner: inner{X: 1, Y: 2}, Z: 3},
		nilly{},
		withBytes{B: []byte("hi"), A4: [4]byte{1, 2, 3, 4}},
	}
	for _, v := range values {
		blob, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("Marshal(%T): %v", v, err)
		}
		ours := reflect.New(reflect.TypeOf(v))
		if err := json.Unmarshal(blob, ours.Interface()); err != nil {
			t.Fatalf("Unmarshal(%T): %v", v, err)
		}
		std := reflect.New(reflect.TypeOf(v))
		if err := jsonv2.Unmarshal(blob, std.Interface()); err != nil {
			t.Fatalf("encoding/json/v2 Unmarshal(%T): %v", v, err)
		}
		if !reflect.DeepEqual(ours.Interface(), std.Interface()) {
			t.Errorf("round trip diverged for %T:\n  encoding/json/v2: %#v\n  go.gh.ink/json/v2: %#v",
				v, std.Interface(), ours.Interface())
		}
	}
}

func TestPreheat(t *testing.T) {
	if err := json.Preheat(reflect.TypeFor[basic]()); err != nil {
		t.Errorf("Preheat: %v", err)
	}
	if err := json.PreheatMany([]reflect.Type{reflect.TypeFor[basic](), reflect.TypeFor[embeds]()}); err != nil {
		t.Errorf("PreheatMany: %v", err)
	}
}
