package llm

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/invopop/jsonschema"
)

// schemaFor builds the JSON schema of out's type for structured output.
// Pointer fields become "T or null" (the library alone would make them required non-null). It returns nil when out is not a struct
// pointer, and the caller then asks for plain JSON.
func schemaFor(out any) map[string]any {
	t := reflect.TypeOf(out)
	if t == nil || t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return nil
	}
	r := &jsonschema.Reflector{DoNotReference: true, ExpandedStruct: true}
	b, err := json.Marshal(r.ReflectFromType(t.Elem()))
	if err != nil {
		return nil
	}
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return nil
	}
	stripMeta(m)
	nullable(t.Elem(), m)
	return m
}

// stripMeta removes $schema and $id, which providers reject.
func stripMeta(v any) {
	switch x := v.(type) {
	case map[string]any:
		delete(x, "$schema")
		delete(x, "$id")
		for _, c := range x {
			stripMeta(c)
		}
	case []any:
		for _, c := range x {
			stripMeta(c)
		}
	}
}

// nullable walks t and its schema together and wraps every pointer field as
// anyOf [T, null], so a model may answer null where Go has a nil pointer.
func nullable(t reflect.Type, s map[string]any) {
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
		t = t.Elem()
	}
	if t.Kind() == reflect.Slice {
		return
	}
	if t.Kind() != reflect.Struct {
		return
	}
	props, _ := s["properties"].(map[string]any)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name := strings.Split(f.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		fs, ok := props[name].(map[string]any)
		if !ok {
			continue
		}
		ft := f.Type
		isPtr := ft.Kind() == reflect.Ptr
		inner := fs
		if items, ok := fs["items"].(map[string]any); ok {
			inner = items
		}
		nullable(ft, inner)
		if isPtr {
			props[name] = map[string]any{"anyOf": []any{fs, map[string]any{"type": "null"}}}
		}
	}
}
