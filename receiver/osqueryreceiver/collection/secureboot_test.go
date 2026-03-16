package collection

import (
	"reflect"
	"testing"
)

func TestSecureBootCollection_Metadata(t *testing.T) {
	c := SecureBootCollection{}

	if got := c.GetName(); got != SecureBootCollectionName {
		t.Fatalf("expected name %q but got %q", SecureBootCollectionName, got)
	}

	if got := c.GetQuery(); got != SecureBootCollectionQuery {
		t.Fatalf("expected query %q but got %q", SecureBootCollectionQuery, got)
	}
}

func TestSecureBootCollection_Unmarshal(t *testing.T) {
	col := SecureBootCollection{}

	tests := []struct {
		name  string
		input any
		want  any
	}{
		{
			name: "full payload",
			input: map[string]any{
				"description":       "  secure boot enabled  ",
				"secure_boot":       1,
				"setup_mode":        "2",
				"secure_mode":       float64(3),
				"kernel_extensions": 4,
				"mdm_operations":    "5",
			},
			want: map[string]any{
				"description":       "secure boot enabled",
				"secure_boot":       float64(1),
				"setup_mode":        float64(2),
				"secure_mode":       float64(3),
				"kernel_extensions": float64(4),
				"mdm_operations":    float64(5),
			},
		},
		{
			name:  "non map input",
			input: []int{1, 2, 3},
			want:  nil,
		},
		{
			name: "zero values removed",
			input: map[string]any{
				"description":       "   ",
				"secure_boot":       0,
				"setup_mode":        0,
				"secure_mode":       0,
				"kernel_extensions": 0,
				"mdm_operations":    0,
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := col.Unmarshal(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("unexpected unmarshal result: got %#v want %#v", got, tt.want)
			}
		})
	}
}
