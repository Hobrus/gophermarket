package luhn

import "testing"

func TestIsValid(t *testing.T) {
	tests := []struct {
		num      string
		expected bool
	}{
		{"79927398713", true},
		{"12345678903", true},
		{"123", false},
		{"79927398714", false},
	}

	for _, tt := range tests {
		if IsValid(tt.num) != tt.expected {
			t.Errorf("IsValid(%s) expected %v", tt.num, tt.expected)
		}
	}
}
