package marshaller

import (
	"encoding"
	stdjson "encoding/json"
	"reflect"
	"strings"
	"sync"
)

// The backend is fast but not a perfect stand-in for encoding/json. Rather than
// let a difference reach the caller, values that would hit one are encoded or
// decoded by encoding/json instead. The checks below decide that from the type
// alone, so the answer is computed once per type and cached.
//
// Deferring to encoding/json rather than emulating it also means this package
// tracks whatever the toolchain's encoding/json does. Go 1.27 rebuilt it on top
// of encoding/json/v2 and changed several behaviours in the process; nothing
// here names those changes, so it follows them, and whatever a later release
// changes next.

type quirks uint8

const (
	// quirkOmitZero marks a type whose tree uses the "omitzero" tag option.
	quirkOmitZero quirks = 1 << iota
	// quirkMapKey marks a type whose tree contains a map whose key type the
	// backend does not handle the way encoding/json does.
	quirkMapKey
	// quirkStringOpt marks a type whose tree uses the "string" tag option. The
	// two write such a field identically but read it differently: the backend
	// is content with a quoted body encoding/json rejects, and truncates
	// "1.5" into an int rather than refusing it. Decoding is diverted, encoding
	// is not.
	quirkStringOpt
)

var (
	textMarshalerType   = reflect.TypeFor[encoding.TextMarshaler]()
	textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()
	jsonMarshalerType   = reflect.TypeFor[stdjson.Marshaler]()
)

// quirksCache memoises typeQuirks so each type is walked at most once per
// process. It maps reflect.Type to quirks.
var quirksCache sync.Map

// typeQuirks reports which backend shortcomings a value of type t could run
// into.
//
// Interface-typed fields are not followed: their dynamic type is only known at
// call time, so a problem reachable solely through an interface is missed. A
// map keyed by an interface is itself diverted, since that is the case where an
// unknown dynamic type decides the encoding of a member name.
func typeQuirks(t reflect.Type) quirks {
	if t == nil {
		return 0
	}
	if cached, ok := quirksCache.Load(t); ok {
		return cached.(quirks)
	}
	found := walkType(t, make(map[reflect.Type]bool))
	quirksCache.Store(t, found)
	return found
}

// walkType traverses the type tree rooted at t. seen breaks cycles in recursive
// types: re-entering a type already on the stack cannot expose anything the
// outer visit will not reach by itself.
func walkType(t reflect.Type, seen map[reflect.Type]bool) quirks {
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return walkType(t.Elem(), seen)
	case reflect.Map:
		found := walkType(t.Elem(), seen)
		if !safeMapKey(t.Key()) {
			found |= quirkMapKey
		}
		return found
	case reflect.Struct:
		if seen[t] {
			return 0
		}
		seen[t] = true
		// A type carrying its own MarshalJSON is emitted through that method
		// and never has its fields inspected, so nothing below it matters.
		if t.Implements(jsonMarshalerType) {
			return 0
		}
		var found quirks
		for i := range t.NumField() {
			found |= walkField(t.Field(i), seen)
		}
		return found
	}
	return 0
}

// walkField reports what f contributes, including the "omitzero" option when f
// carries it directly.
func walkField(f reflect.StructField, seen map[reflect.Type]bool) quirks {
	tag := f.Tag.Get("json")
	if tag == "-" {
		// The encoder drops the field outright, along with anything under it.
		return 0
	}
	if !f.IsExported() && !isEmbeddedStruct(f) {
		// Unexported fields are invisible to the encoder, the one exception
		// being embedded structs, which it still walks for exported fields.
		return 0
	}
	found := walkType(f.Type, seen)
	if _, opts, tagged := strings.Cut(tag, ","); tagged {
		for opt := range strings.SplitSeq(opts, ",") {
			switch opt {
			case "omitzero":
				found |= quirkOmitZero
			case "string":
				found |= quirkStringOpt
			}
		}
	}
	return found
}

// isEmbeddedStruct reports whether f is an embedded struct or a pointer to one.
func isEmbeddedStruct(f reflect.StructField) bool {
	if !f.Anonymous {
		return false
	}
	t := f.Type
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Kind() == reflect.Struct
}

// safeMapKey reports whether the backend turns keys of type t into object
// member names exactly as encoding/json does.
//
// The list is a whitelist because the ways of disagreeing are open-ended and
// the consequences are not symmetric: a needless detour to encoding/json only
// costs speed, while a wrong guess ships different bytes. Everything on it was
// checked against both Go 1.26 and Go 1.27, whose encoding/json packages differ
// here. Notably a pointer key makes the backend emit an unquoted member name,
// which is not valid JSON at all, and float keys are rejected by the backend
// but accepted from Go 1.27 on.
func safeMapKey(t reflect.Type) bool {
	if t.Implements(textMarshalerType) || reflect.PointerTo(t).Implements(textMarshalerType) ||
		t.Implements(textUnmarshalerType) || reflect.PointerTo(t).Implements(textUnmarshalerType) {
		return true
	}
	switch t.Kind() {
	case reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	}
	return false
}
