package collection

import (
	"reflect"
	"testing"
)

func TestOSInfoCollection_Metadata(t *testing.T) {
	col := OSInfoCollection{}

	if got := col.GetName(); got != OSInfoCollectionName {
		t.Fatalf("expected name %q but got %q", OSInfoCollectionName, got)
	}

	if got := col.GetQuery(); got != OSInfoCollectionQuery {
		t.Fatalf("expected query %q but got %q", OSInfoCollectionQuery, got)
	}
}

func TestOSInfoCollection_Unmarshal(t *testing.T) {
	col := OSInfoCollection{}

	tests := []struct {
		name string
		in   any
		want any
	}{
		{
			name: "full payload",
			in: map[string]any{
				"name":          " macOS ",
				"version":       " 14.0 ",
				"build":         " 23A ",
				"platform":      " darwin ",
				"platform_like": " unix ",
				"codename":      " Sonoma ",
				"arch":          " arm64 ",
			},
			want: map[string]any{
				"name":          "macOS",
				"version":       "14.0",
				"build":         "23A",
				"platform":      "darwin",
				"platform_like": "unix",
				"codename":      "Sonoma",
				"arch":          "arm64",
			},
		},
		{
			name: "non map input",
			in:   []string{"invalid"},
			want: nil,
		},
		{
			name: "empty after sanitize",
			in: map[string]any{
				"name":    " ",
				"version": "",
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := col.Unmarshal(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("unexpected unmarshal result: got %#v want %#v", got, tt.want)
			}
		})
	}
}
