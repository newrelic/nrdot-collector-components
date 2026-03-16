package collection

import (
	"testing"
)

func TestUserCollection_GetName(t *testing.T) {
	uc := NewUserCollection()
	expectedName := "users_info"
	if uc.GetName() != expectedName {
		t.Errorf("expected name %s, got %s", expectedName, uc.GetName())
	}
}

func TestUserCollection_GetQuery(t *testing.T) {
	uc := NewUserCollection()
	query := uc.GetQuery()
	if query == "" {
		t.Error("expected non-empty query")
	}
}

func TestUserCollection_Unmarshal(t *testing.T) {
	uc := NewUserCollection()

	// Test with valid input
	input := []map[string]any{
		{"username": "user1", "groups": "group1, group2"},
		{"username": "user2", "groups": "group3"},
	}
	result := uc.Unmarshal(input)
	if result == nil {
		t.Error("expected non-nil result for valid input")
	}

	// Test with empty input
	emptyInput := []map[string]any{}
	result = uc.Unmarshal(emptyInput)
	if result != nil {
		t.Error("expected nil result for empty input")
	}

	// Test with invalid input type
	invalidInput := "invalid"
	result = uc.Unmarshal(invalidInput)
	if result != nil {
		t.Error("expected nil result for invalid input type")
	}
}
