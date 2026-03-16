package collection

import (
	"reflect"
	"testing"
)

func TestSystemInfoCollection_Metadata(t *testing.T) {
	col := SystemInfoCollection{}

	if got := col.GetName(); got != SystemInfoCollectionName {
		t.Fatalf("expected name %q but got %q", SystemInfoCollectionName, got)
	}

	if got := col.GetQuery(); got != SystemInfoCollectionQuery {
		t.Fatalf("expected query %q but got %q", SystemInfoCollectionQuery, got)
	}
}

func TestSystemInfoCollection_Unmarshal(t *testing.T) {
	col := SystemInfoCollection{}

	tests := []struct {
		name string
		in   any
		want any
	}{
		{
			name: "full payload",
			in: map[string]any{
				"hostname":           " host ",
				"uuid":               "uuid-123 ",
				"cpu_type":           " type",
				"cpu_subtype":        " subtype ",
				"cpu_brand":          " brand ",
				"cpu_physical_cores": 4,
				"cpu_logical_cores":  8.0,
				"physical_memory":    " 16GB ",
				"hardware_vendor":    " vendor ",
				"hardware_model":     " model ",
				"computer_name":      " host.local ",
				"emulated_cpu_type":  " Rosetta ",
			},
			want: map[string]any{
				"hostname":           "host",
				"uuid":               "uuid-123",
				"cpu_type":           "type",
				"cpu_subtype":        "subtype",
				"cpu_brand":          "brand",
				"cpu_physical_cores": float64(4),
				"cpu_logical_cores":  float64(8),
				"physical_memory":    "16GB",
				"hardware_vendor":    "vendor",
				"hardware_model":     "model",
				"computer_name":      "host.local",
				"emulated_cpu_type":  "Rosetta",
			},
		},
		{
			name: "non map input",
			in:   []int{1, 2},
			want: nil,
		},
		{
			name: "empty after sanitize",
			in: map[string]any{
				"hostname":           " ",
				"cpu_physical_cores": 0,
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
