package json_test

import (
	stdjson "encoding/json"
	"reflect"
	"testing"

	"go.gh.ink/json"
)

// The case tables in conformance_test.go only reach the differences someone
// thought to write down. Several of the ones already found depend on the value
// rather than the type, a float exponent or a stray byte in a string, and those
// are exactly what a table misses. These targets generate values instead.

var fuzzSeeds = []string{
	`{"a":1,"b":"x","c":true}`,
	`{"a":1e-7,"b":1e-8,"c":1e-9}`,
	`{"a":1e-10,"b":1e21,"c":-0}`,
	`[1.5,2e-7,3e300,4e-300]`,
	`{"s":"héllo 世界 🎉","t":"<b>&\"q\"</b>"}`,
	`{"s":"a\u0000b\u001fc"}`,
	`{"s":"\ud83c\udf89","u":"\ud800"}`,
	`{"nested":{"deep":{"deeper":[1,{"x":1e-8}]}}}`,
	`{"big":12345678901234567890123,"small":1.0000000000000000000000001}`,
	`[[],{},null,true,false,0,-0,"",[[[]]]]`,
	"{\"s\":\"a\xffb\"}",
	`1e-7`,
	`"plain"`,
	`null`,
}

// FuzzMarshalConformance decodes the input into a generic value and re-encodes
// it through both implementations. Going through a decode first is what makes
// the generated floats and strings realistic; encoding one straight from fuzz
// bytes would mostly produce values no encoder ever sees.
func FuzzMarshalConformance(f *testing.F) {
	for _, s := range fuzzSeeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var v any
		if err := stdjson.Unmarshal(data, &v); err != nil {
			t.Skip()
		}

		want, wantErr := stdjson.Marshal(v)
		got, gotErr := json.Marshal(v)

		if (wantErr != nil) != (gotErr != nil) {
			t.Fatalf("error mismatch on %q:\n  encoding/json: %v\n  go.gh.ink/json: %v", data, wantErr, gotErr)
		}
		if wantErr != nil {
			return
		}
		if string(got) != string(want) && !sameJSON(t, got, want) {
			t.Errorf("output mismatch on %q:\n  encoding/json: %s\n  go.gh.ink/json: %s", data, want, got)
		}
	})
}

// FuzzUnmarshalConformance decodes the input into each of a handful of shapes
// and compares what lands there. Malformed input is on purpose: both sides have
// to reject the same documents.
func FuzzUnmarshalConformance(f *testing.F) {
	for _, s := range fuzzSeeds {
		f.Add([]byte(s))
	}

	targets := []func() any{
		func() any { return new(any) },
		func() any { return new(basic) },
		func() any { return new(omitted) },
		func() any { return new(stringOpt) },
		func() any { return new(embedsValue) },
		func() any { return new(withBytes) },
		func() any { return new(withNumber) },
		func() any { return new(withInterface) },
		func() any { return new(map[string]any) },
		func() any { return new(map[string]float64) },
		func() any { return new([]float64) },
		func() any { return new(float64) },
		func() any { return new(string) },
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// The backend accepts some documents encoding/json rejects, which the
		// library documents rather than pays to prevent, so a malformed one
		// cannot be held to matching. What still has to hold is that any error
		// it does report reads like encoding/json's.
		wellFormed := stdjson.Valid(data)

		for i, newV := range targets {
			wantV, gotV := newV(), newV()
			wantErr := stdjson.Unmarshal(data, wantV)
			gotErr := json.Unmarshal(data, gotV)

			if !wellFormed {
				if gotErr != nil && wantErr != nil && gotErr.Error() != wantErr.Error() {
					t.Errorf("target %d: error message mismatch on %q:\n  encoding/json: %v\n  go.gh.ink/json: %v",
						i, data, wantErr, gotErr)
				}
				continue
			}

			if (wantErr != nil) != (gotErr != nil) {
				t.Fatalf("target %d: error mismatch on %q:\n  encoding/json: %v\n  go.gh.ink/json: %v",
					i, data, wantErr, gotErr)
			}
			if wantErr != nil {
				// The target is documented as unspecified past an error, so
				// only the message is compared here.
				if gotErr.Error() != wantErr.Error() {
					t.Errorf("target %d: error message mismatch on %q:\n  encoding/json: %v\n  go.gh.ink/json: %v",
						i, data, wantErr, gotErr)
				}
				continue
			}
			if !reflect.DeepEqual(wantV, gotV) {
				t.Errorf("target %d: decoded state mismatch on %q:\n  encoding/json: %#v\n  go.gh.ink/json: %#v",
					i, data, wantV, gotV)
			}
		}
	})
}

// FuzzRoundTrip checks that what this library writes, it reads back the way
// encoding/json would.
func FuzzRoundTrip(f *testing.F) {
	for _, s := range fuzzSeeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var v any
		if err := stdjson.Unmarshal(data, &v); err != nil {
			t.Skip()
		}
		blob, err := json.Marshal(v)
		if err != nil {
			t.Skip()
		}

		var ours, std any
		oursErr := json.Unmarshal(blob, &ours)
		stdErr := stdjson.Unmarshal(blob, &std)
		if (oursErr != nil) != (stdErr != nil) {
			t.Fatalf("re-decode error mismatch on %q:\n  encoding/json: %v\n  go.gh.ink/json: %v", blob, stdErr, oursErr)
		}
		if stdErr != nil {
			return
		}
		if !reflect.DeepEqual(ours, std) {
			t.Errorf("re-decode mismatch on %q:\n  encoding/json: %#v\n  go.gh.ink/json: %#v", blob, std, ours)
		}
	})
}
