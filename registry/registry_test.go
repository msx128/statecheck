package registry

import (
	"testing"
)

func TestCheckResponceCode(t *testing.T) {
	tests := []struct {
		name string
		code int
		want bool
	}{
		{"less than 200", 199, false},
		{"more than 299", 300, false},
		{"ok code(200..300)", 299, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkResponseCode(tt.code)
			got := err == nil
			if got != tt.want {
				t.Errorf("checkResponseCode(%d); returns error: %t; want %t", tt.code, got, tt.want)
			}
		})
	}
}

// I don't want to test idk
