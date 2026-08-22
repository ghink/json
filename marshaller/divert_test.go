package marshaller

import (
	stdjson "encoding/json"
	"reflect"
	"testing"
	"time"

	gojson "github.com/goccy/go-json"
)

type plain struct {
	A int
	B string
}

type omitEmpty struct {
	A int `json:"a,omitempty"`
}

// The tag names the field "omitzero"; it does not set the option.
type nameIsOmitZero struct {
	A int `json:"omitzero"`
}

type tagged struct {
	A int `json:"a,omitzero"`
}

type taggedNoName struct {
	A int `json:",omitzero"`
}

type nestedValue struct{ Inner tagged }

type nestedPointer struct{ Inner *tagged }

type nestedSlice struct{ Items []tagged }

type nestedArray struct{ Items [2]tagged }

type nestedMapValue struct{ M map[string]tagged }

type keyed string

type keyedMap struct{ M map[keyed]int }

type recursivePlain struct {
	Next *recursivePlain
	A    int
}

type recursiveTagged struct {
	Next *recursiveTagged
	A    int `json:"a,omitzero"`
}

type skipped struct {
	Inner tagged `json:"-"`
}

// The encoder ignores unexported non-embedded fields, so the tag under inner is
// unreachable.
type hiddenField struct {
	inner tagged
	A     int
}

// The encoder does walk embedded structs of unexported type for their exported
// fields, so the tag under tagged is reachable and A is promoted.
type embedsUnexported struct {
	tagged
	B int
}

type selfMarshaling struct {
	A int `json:"a,omitzero"`
}

func (selfMarshaling) MarshalJSON() ([]byte, error) { return []byte(`"fixed"`), nil }

type wrapsSelfMarshaling struct{ M selfMarshaling }

type behindInterface struct{ V any }

func TestDivertOmitZero(t *testing.T) {
	cases := []struct {
		value any
		want  bool
	}{
		{nil, false},
		{0, false},
		{plain{}, false},
		{omitEmpty{}, false},
		{nameIsOmitZero{}, false},
		{time.Time{}, false},
		{stdjson.RawMessage{}, false},
		{tagged{}, true},
		{&tagged{}, true},
		{taggedNoName{}, true},
		{[]tagged{}, true},
		{[3]tagged{}, true},
		{map[string]tagged{}, true},
		// A map key becomes a member name through MarshalText or through its
		// basic kind, so tags inside a key type never reach the output. This
		// one is still diverted, by quirkMapKey, since a plain struct is not
		// a key type the backend can be trusted with.
		{map[tagged]int{}, false},
		{nestedValue{}, true},
		{nestedPointer{}, true},
		{nestedSlice{}, true},
		{nestedArray{}, true},
		{nestedMapValue{}, true},
		{keyedMap{}, false},
		{recursivePlain{}, false},
		{recursiveTagged{}, true},
		{skipped{}, false},
		{hiddenField{}, false},
		{embedsUnexported{}, true},
		{selfMarshaling{}, false},
		{wrapsSelfMarshaling{}, false},
		// The dynamic type behind an interface field is unknown until encode
		// time, so the walk cannot see the tag inside it.
		{behindInterface{V: tagged{}}, false},
	}

	for _, c := range cases {
		vt := reflect.TypeOf(c.value)
		if got := typeQuirks(vt)&quirkOmitZero != 0; got != c.want {
			t.Errorf("quirkOmitZero(%v) = %v, want %v", vt, got, c.want)
		}
		// A second call must come from the cache and agree with the first.
		if got := typeQuirks(vt)&quirkOmitZero != 0; got != c.want {
			t.Errorf("quirkOmitZero(%v) cached = %v, want %v", vt, got, c.want)
		}
	}
}

type textKey struct{ S string }

func (t textKey) MarshalText() ([]byte, error)  { return []byte("T:" + t.S), nil }
func (t *textKey) UnmarshalText(b []byte) error { t.S = string(b); return nil }

type namedString string

type namedInt int

type plainKeyed struct{ M map[string]int }

type floatKeyed struct{ M map[float64]int }

type deepFloatKeyed struct{ Items []map[float32]string }

type ptrKeyed struct{ M map[*int]int }

// TestDivertMapKey checks the whitelist that decides which key types the
// backend may handle. Only string, integer and text-marshalling keys stay on
// it; everything else is sent to encoding/json, whose treatment of the rest
// changed in Go 1.27 and may change again.
func TestDivertMapKey(t *testing.T) {
	cases := []struct {
		value any
		want  bool
	}{
		{nil, false},
		{plain{}, false},
		{map[string]int{}, false},
		{map[namedString]int{}, false},
		{map[int]string{}, false},
		{map[int8]string{}, false},
		{map[uint64]string{}, false},
		{map[namedInt]string{}, false},
		{map[textKey]int{}, false},
		{plainKeyed{}, false},

		{map[float64]int{}, true},
		{map[float32]int{}, true},
		{map[bool]int{}, true},
		{map[*int]int{}, true},
		{map[[2]int]int{}, true},
		{map[any]int{}, true},
		{map[complex128]int{}, true},
		{map[plain]int{}, true},
		{floatKeyed{}, true},
		{deepFloatKeyed{}, true},
		{ptrKeyed{}, true},
		{[]map[bool]int{}, true},
		{&floatKeyed{}, true},
		{map[string]map[float64]int{}, true},
	}

	for _, c := range cases {
		vt := reflect.TypeOf(c.value)
		if got := typeQuirks(vt)&quirkMapKey != 0; got != c.want {
			t.Errorf("quirkMapKey(%v) = %v, want %v", vt, got, c.want)
		}
	}
}

// TestMarshalMatchesStdForOmitZero pins the whole point of the diversion: on
// every backend, a type carrying "omitzero" must encode the way encoding/json
// encodes it.
func TestMarshalMatchesStdForOmitZero(t *testing.T) {
	cases := []any{
		tagged{},
		tagged{A: 7},
		&tagged{},
		taggedNoName{},
		[]tagged{{}, {A: 1}},
		nestedValue{},
		nestedValue{Inner: tagged{A: 2}},
		nestedPointer{},
		nestedPointer{Inner: &tagged{}},
		nestedSlice{Items: []tagged{{}}},
		nestedArray{},
		recursiveTagged{Next: &recursiveTagged{A: 1}},
		skipped{Inner: tagged{A: 3}},
		hiddenField{A: 4},
		embedsUnexported{B: 5},
		embedsUnexported{tagged: tagged{A: 6}, B: 7},
		wrapsSelfMarshaling{},
		plain{A: 1, B: "b"},
		omitEmpty{},
	}

	for _, v := range cases {
		want, err := stdjson.Marshal(v)
		if err != nil {
			t.Fatalf("encoding/json.Marshal(%#v): %v", v, err)
		}
		got, err := Marshal(v)
		if err != nil {
			t.Fatalf("Marshal(%#v): %v", v, err)
		}
		if string(got) != string(want) {
			t.Errorf("Marshal(%#v) = %s, want %s", v, got, want)
		}
	}
}

func TestUnmarshalRoundTrip(t *testing.T) {
	var got tagged
	if err := Unmarshal([]byte(`{"a":9}`), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.A != 9 {
		t.Errorf("got %+v, want A=9", got)
	}

	// "omitzero" is encode-only, so an absent key must simply leave the zero
	// value in place rather than upset the decoder.
	got = tagged{A: 1}
	if err := Unmarshal([]byte(`{}`), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.A != 1 {
		t.Errorf("got %+v, want A untouched", got)
	}
}

func TestPreheat(t *testing.T) {
	if err := Preheat(reflect.TypeFor[tagged]()); err != nil {
		t.Errorf("Preheat: %v", err)
	}
	if err := PreheatMany([]reflect.Type{reflect.TypeFor[plain](), reflect.TypeFor[nestedValue]()}); err != nil {
		t.Errorf("PreheatMany: %v", err)
	}
}

// TestGoJSONStillLacksOmitZero is a tripwire on the upstream gap that forces
// the diversion in the first place. When it fails, goccy/go-json has grown
// "omitzero" support: set backendSupportsOmitZero to true in go-json.go and
// drop this test.
func TestGoJSONStillLacksOmitZero(t *testing.T) {
	out, err := gojson.Marshal(tagged{})
	if err != nil {
		t.Fatalf("go-json Marshal: %v", err)
	}
	if string(out) != `{"a":0}` {
		t.Errorf("go-json now emits %s for a zero omitzero field; it appears to support omitzero, so backendSupportsOmitZero can be flipped", out)
	}
}
