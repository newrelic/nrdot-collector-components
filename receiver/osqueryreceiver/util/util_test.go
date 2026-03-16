package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContains(t *testing.T) {
	tests := map[string]struct {
		slice    []string
		item     string
		expected bool
	}{
		"item present": {
			slice:    []string{"apple", "banana", "cherry"},
			item:     "banana",
			expected: true,
		},
		"item absent": {
			slice:    []string{"apple", "banana", "cherry"},
			item:     "date",
			expected: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(tt *testing.T) {
			result := Contains(tc.slice, tc.item)
			assert.Equal(tt, tc.expected, result)
		})
	}
}

func TestGetString(t *testing.T) {
	tests := map[string]struct {
		obj      map[string]any
		item     string
		expected string
	}{
		"item present": {
			obj:      map[string]any{"apple": "red", "banana": "yellow", "cherry": "red"},
			item:     "banana",
			expected: "yellow",
		},
		"item absent": {
			obj:      map[string]any{"apple": "red", "banana": "yellow", "cherry": "red"},
			item:     "date",
			expected: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(tt *testing.T) {
			result := GetString(tc.obj, tc.item)
			assert.Equal(tt, tc.expected, result)
		})
	}
}

func TestGetInt(t *testing.T) {
	tests := map[string]struct {
		obj      map[string]any
		item     string
		expected int
	}{
		"item present - string": {
			obj:      map[string]any{"apple": "red", "banana": "10", "cherry": "red"},
			item:     "banana",
			expected: 10,
		},
		"item present - float64": {
			obj:      map[string]any{"apple": "red", "banana": 20.0, "cherry": "red"},
			item:     "banana",
			expected: 20,
		},
		"item present - int": {
			obj:      map[string]any{"apple": "red", "banana": 30, "cherry": "red"},
			item:     "banana",
			expected: 30,
		},
		"item absent": {
			obj:      map[string]any{"apple": "red", "banana": "yellow", "cherry": "red"},
			item:     "date",
			expected: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(tt *testing.T) {
			result := GetInt(tc.obj, tc.item)
			assert.Equal(tt, tc.expected, result)
		})
	}
}

func TestGetInt64(t *testing.T) {
	tests := map[string]struct {
		obj      map[string]any
		item     string
		expected int64
	}{
		"item present - string": {
			obj:      map[string]any{"apple": "red", "banana": "100", "cherry": "red"},
			item:     "banana",
			expected: 100,
		},
		"item present - float64": {
			obj:      map[string]any{"apple": "red", "banana": 200.0, "cherry": "red"},
			item:     "banana",
			expected: 200,
		},
		"item present - int": {
			obj:      map[string]any{"apple": "red", "banana": 300, "cherry": "red"},
			item:     "banana",
			expected: 300,
		},
		"item present - int64": {
			obj:      map[string]any{"apple": "red", "banana": int64(400), "cherry": "red"},
			item:     "banana",
			expected: 400,
		},
		"item absent": {
			obj:      map[string]any{"apple": "red", "banana": "yellow", "cherry": "red"},
			item:     "date",
			expected: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(tt *testing.T) {
			result := GetInt64(tc.obj, tc.item)
			assert.Equal(tt, tc.expected, result)
		})
	}
}
