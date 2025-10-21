package main

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		want         string
	}{
		{
			name:         "returns env value when set",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "actual",
			want:         "actual",
		},
		{
			name:         "returns default when env not set",
			key:          "UNSET_KEY",
			defaultValue: "default",
			envValue:     "",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				if err := os.Setenv(tt.key, tt.envValue); err != nil {
					t.Fatalf("failed to set env var: %v", err)
				}
				defer func() {
					_ = os.Unsetenv(tt.key)
				}()
			}

			got := getEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		want         int
	}{
		{
			name:         "returns parsed int when valid",
			key:          "TEST_INT",
			defaultValue: 42,
			envValue:     "100",
			want:         100,
		},
		{
			name:         "returns default when not set",
			key:          "UNSET_INT",
			defaultValue: 42,
			envValue:     "",
			want:         42,
		},
		{
			name:         "returns default when invalid int",
			key:          "INVALID_INT",
			defaultValue: 42,
			envValue:     "not-a-number",
			want:         42,
		},
		{
			name:         "handles zero value",
			key:          "ZERO_INT",
			defaultValue: 42,
			envValue:     "0",
			want:         0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				if err := os.Setenv(tt.key, tt.envValue); err != nil {
					t.Fatalf("failed to set env var: %v", err)
				}
				defer func() {
					_ = os.Unsetenv(tt.key)
				}()
			}

			got := getEnvInt(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnvInt() = %v, want %v", got, tt.want)
			}
		})
	}
}
