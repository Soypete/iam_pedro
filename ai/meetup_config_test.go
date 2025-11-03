package ai

import (
	"testing"
)

func TestLoadStreamConfig(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		wantErr    bool
	}{
		{
			name:       "valid config - full path",
			configPath: "../configs/streams/golang-nov-2025.yaml",
			wantErr:    false,
		},
		{
			name:       "valid config - live coding template",
			configPath: "../configs/streams/live-coding-template.yaml",
			wantErr:    false,
		},
		{
			name:       "empty path",
			configPath: "",
			wantErr:    true,
		},
		{
			name:       "non-existent file",
			configPath: "../configs/streams/nonexistent.yaml",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadStreamConfig(tt.configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadStreamConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && config == nil {
				t.Error("LoadStreamConfig() returned nil config when error not expected")
			}
			if !tt.wantErr {
				// Validate the config loaded correctly
				if config.EventInfo.Title == "" {
					t.Error("Config loaded but event title is empty")
				}
				if config.Metadata.Name == "" {
					t.Error("Config loaded but metadata name is empty")
				}
			}
		})
	}
}

func TestLoadMeetupConfig(t *testing.T) {
	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{
			name:    "valid config - golang-nov-2025",
			slug:    "golang-nov-2025",
			wantErr: false,
		},
		{
			name:    "valid config - live-coding-template",
			slug:    "live-coding-template",
			wantErr: false,
		},
		{
			name:    "empty slug",
			slug:    "",
			wantErr: true,
		},
		{
			name:    "non-existent config",
			slug:    "nonexistent-meetup",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadMeetupConfig(tt.slug)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadMeetupConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && config == nil {
				t.Error("LoadMeetupConfig() returned nil config when error not expected")
			}
			if !tt.wantErr {
				// Validate the config loaded correctly
				if config.EventInfo.Title == "" {
					t.Error("Config loaded but event title is empty")
				}
				if config.Metadata.Name == "" {
					t.Error("Config loaded but metadata name is empty")
				}
			}
		})
	}
}

func TestGenerateMeetupAddendum(t *testing.T) {
	// Load the golang-nov-2025 config
	config, err := LoadMeetupConfig("golang-nov-2025")
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	addendum := GenerateMeetupAddendum(config)

	// Check that key elements are present
	if addendum == "" {
		t.Error("Generated addendum is empty")
	}

	// Check for key content
	expectedStrings := []string{
		"SPECIAL EVENT",
		"Beyond Hello World",
		"Miriah Peterson",
		"Forge Utah",
		"Schedule:",
	}

	for _, expected := range expectedStrings {
		if !contains(addendum, expected) {
			t.Errorf("Addendum missing expected string: %s", expected)
		}
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *MeetupConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &MeetupConfig{
				Metadata: MeetupMetadata{
					Name: "Test Meetup",
				},
				EventInfo: EventInfo{
					Title: "Test Event",
				},
			},
			wantErr: true, // Will fail because date is zero
		},
		{
			name: "missing event title",
			config: &MeetupConfig{
				Metadata: MeetupMetadata{
					Name: "Test Meetup",
				},
				EventInfo: EventInfo{
					Title: "",
				},
			},
			wantErr: true,
		},
		{
			name: "missing metadata name",
			config: &MeetupConfig{
				Metadata: MeetupMetadata{
					Name: "",
				},
				EventInfo: EventInfo{
					Title: "Test Event",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
