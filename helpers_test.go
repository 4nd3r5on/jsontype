package jsontype_test

import (
	"reflect"
	"testing"

	"github.com/4nd3r5on/jsontype"
)

func TestPathEncodeDecode_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		path []string
	}{
		{
			name: "root",
			path: []string{},
		},
		{
			name: "simple object",
			path: []string{"user", "name"},
		},
		{
			name: "array wildcard",
			path: []string{"items", ""},
		},
		{
			name: "array index",
			path: []string{"items", "0"},
		},
		{
			name: "nested array index",
			path: []string{"a", "1", "b", "2"},
		},
		{
			name: "mixed wildcard and index",
			path: []string{"users", "", "posts", "3", "title"},
		},
		{
			name: "deep mixed",
			path: []string{"a", "", "b", "0", "c", "", "d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := jsontype.PathToString(tt.path)
			decoded := jsontype.StringToPath(encoded)

			if !reflect.DeepEqual(decoded, tt.path) {
				t.Fatalf(
					"round-trip mismatch\npath:    %#v\nencoded: %q\ndecoded: %#v",
					tt.path,
					encoded,
					decoded,
				)
			}
		})
	}
}
