package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// MeetupConfig represents the full configuration for a meetup event
type MeetupConfig struct {
	Metadata  MeetupMetadata  `yaml:"metadata"`
	EventInfo EventInfo       `yaml:"event_info"`
	Location  Location        `yaml:"location"`
	Links     Links           `yaml:"links"`
	Schedule  []ScheduleItem  `yaml:"schedule"`
	Speakers  []Speaker       `yaml:"speakers"`
	Hosts     []Host          `yaml:"hosts"`
	ForgeInfo ForgeInfo       `yaml:"forge_info"`
	BotInstructions BotInstructions `yaml:"bot_instructions"`
}

type MeetupMetadata struct {
	Slug            string    `yaml:"slug"`
	Name            string    `yaml:"name"`
	MeetupGroup     string    `yaml:"meetup_group"`
	Date            time.Time `yaml:"date"`
	DurationMinutes int       `yaml:"duration_minutes"`
}

type EventInfo struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

type Location struct {
	Type          string `yaml:"type"` // virtual, hybrid, in-person
	Venue         string `yaml:"venue"`
	Address       string `yaml:"address"`
	Parking       string `yaml:"parking"`
	Accessibility string `yaml:"accessibility"`
}

type Links struct {
	Registration       string `yaml:"registration"`
	VideoCall          string `yaml:"video_call"`
	DialIn             string `yaml:"dial_in"`
	MorePhoneNumbers   string `yaml:"more_phone_numbers"`
}

type ScheduleItem struct {
	Time            string `yaml:"time"`
	Event           string `yaml:"event"`
	DurationMinutes int    `yaml:"duration_minutes"`
	Speaker         string `yaml:"speaker,omitempty"`
}

type Speaker struct {
	Name            string        `yaml:"name"`
	Title           string        `yaml:"title"`
	Handle          string        `yaml:"handle"`
	Bio             string        `yaml:"bio"`
	TalkTitle       string        `yaml:"talk_title"`
	TalkDescription string        `yaml:"talk_description"`
	Social          SocialLinks   `yaml:"social"`
}

type SocialLinks struct {
	GitHub  string `yaml:"github,omitempty"`
	Twitch  string `yaml:"twitch,omitempty"`
	Twitter string `yaml:"twitter,omitempty"`
}

type Host struct {
	Name string `yaml:"name"`
}

type ForgeInfo struct {
	IsForgeEvent   bool     `yaml:"is_forge_event"`
	About          string   `yaml:"about"`
	Mission        string   `yaml:"mission"`
	OtherMeetups   []string `yaml:"other_meetups"`
	CommunityLinks struct {
		Website string `yaml:"website"`
		Slack   string `yaml:"slack,omitempty"`
		Discord string `yaml:"discord,omitempty"`
	} `yaml:"community_links"`
	HowToJoin   string `yaml:"how_to_join"`
	HowToSpeak  string `yaml:"how_to_speak"`
	Sponsorship string `yaml:"sponsorship"`
}

type BotInstructions struct {
	ResponseStyle string   `yaml:"response_style"`
	KeyPoints     []string `yaml:"key_points"`
	Encourage     []string `yaml:"encourage"`
}

// LoadStreamConfig loads a stream configuration from a YAML file path
// This is the primary method for loading stream configs - use an explicit file path.
func LoadStreamConfig(configPath string) (*MeetupConfig, error) {
	if configPath == "" {
		return nil, fmt.Errorf("config path cannot be empty")
	}

	// Read the file directly from the provided path
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read stream config file %s: %w", configPath, err)
	}

	// Parse YAML
	var config MeetupConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse stream config YAML: %w", err)
	}

	// Validate required fields
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid stream config: %w", err)
	}

	return &config, nil
}

// LoadMeetupConfig loads a config by slug (backward compatibility helper)
// Deprecated: Use LoadStreamConfig with full path instead
func LoadMeetupConfig(slug string) (*MeetupConfig, error) {
	if slug == "" {
		return nil, fmt.Errorf("slug cannot be empty")
	}

	// Try multiple possible paths (for tests and backward compatibility)
	possiblePaths := []string{
		filepath.Join("configs", "streams", fmt.Sprintf("%s.yaml", slug)),
		filepath.Join("configs", "meetups", fmt.Sprintf("%s.yaml", slug)),
		filepath.Join("..", "configs", "streams", fmt.Sprintf("%s.yaml", slug)),
		filepath.Join("..", "configs", "meetups", fmt.Sprintf("%s.yaml", slug)),
	}

	for _, path := range possiblePaths {
		config, err := LoadStreamConfig(path)
		if err == nil {
			return config, nil
		}
	}

	return nil, fmt.Errorf("failed to find config for slug '%s' (tried multiple paths)", slug)
}

// validateConfig checks that required fields are present
func validateConfig(config *MeetupConfig) error {
	if config.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}
	if config.EventInfo.Title == "" {
		return fmt.Errorf("event_info.title is required")
	}
	if config.Metadata.Date.IsZero() {
		return fmt.Errorf("metadata.date is required")
	}
	return nil
}

// GenerateMeetupAddendum creates a formatted prompt addendum from the config
func GenerateMeetupAddendum(config *MeetupConfig) string {
	var sb strings.Builder

	// Format the date nicely
	dateStr := config.Metadata.Date.Format("Monday, January 2, 2006 at 3:04 PM MST")

	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("SPECIAL EVENT - %s (%s):\n", config.EventInfo.Title, dateStr))
	sb.WriteString(fmt.Sprintf("We're streaming/discussing the %s meetup!\n\n", config.Metadata.Name))

	// Event details
	sb.WriteString(fmt.Sprintf("Event: %s\n", config.EventInfo.Title))
	if len(config.Speakers) > 0 {
		speaker := config.Speakers[0]
		sb.WriteString(fmt.Sprintf("Speaker: %s - %s\n", speaker.Name, speaker.TalkTitle))
	}
	sb.WriteString(fmt.Sprintf("When: %s\n", dateStr))
	sb.WriteString(fmt.Sprintf("Where: %s", config.Location.Venue))
	if config.Location.Type == "virtual" {
		sb.WriteString(" (Virtual Event)")
	}
	sb.WriteString("\n")

	// Links
	if config.Links.VideoCall != "" {
		sb.WriteString(fmt.Sprintf("Join: %s\n", config.Links.VideoCall))
	}
	if config.Links.Registration != "" {
		sb.WriteString(fmt.Sprintf("Register: %s\n", config.Links.Registration))
	}

	// Schedule
	if len(config.Schedule) > 0 {
		sb.WriteString("\nSchedule:\n")
		for _, item := range config.Schedule {
			if item.Speaker != "" {
				sb.WriteString(fmt.Sprintf("%s - %s (Speaker: %s)\n", item.Time, item.Event, item.Speaker))
			} else {
				sb.WriteString(fmt.Sprintf("%s - %s\n", item.Time, item.Event))
			}
		}
	}

	// Speaker bio (first speaker)
	if len(config.Speakers) > 0 {
		speaker := config.Speakers[0]
		sb.WriteString("\nAbout the Speaker:\n")
		sb.WriteString(fmt.Sprintf("%s\n", speaker.Bio))
	}

	// Forge Utah info
	if config.ForgeInfo.IsForgeEvent {
		sb.WriteString("\nForge Utah Foundation:\n")
		sb.WriteString(fmt.Sprintf("%s\n", config.ForgeInfo.About))
		if config.ForgeInfo.Mission != "" {
			sb.WriteString(fmt.Sprintf("Mission: %s\n", config.ForgeInfo.Mission))
		}
		if len(config.ForgeInfo.OtherMeetups) > 0 {
			sb.WriteString(fmt.Sprintf("We also run: %s\n", strings.Join(config.ForgeInfo.OtherMeetups, ", ")))
		}

		sb.WriteString("\nHow to Get Involved:\n")
		if config.ForgeInfo.HowToJoin != "" {
			sb.WriteString(fmt.Sprintf("- Participate: %s\n", config.ForgeInfo.HowToJoin))
		}
		if config.ForgeInfo.HowToSpeak != "" {
			sb.WriteString(fmt.Sprintf("- Speak: %s\n", config.ForgeInfo.HowToSpeak))
		}
		if config.ForgeInfo.Sponsorship != "" {
			sb.WriteString(fmt.Sprintf("- Sponsor: %s\n", config.ForgeInfo.Sponsorship))
		}
	}

	// Bot instructions
	sb.WriteString("\nWhen viewers ask about:\n")
	sb.WriteString("- The event: Share title, speaker, time, and registration link enthusiastically\n")
	if config.ForgeInfo.IsForgeEvent {
		sb.WriteString("- Forge Utah: Explain our mission and other meetups\n")
	}
	sb.WriteString("- Speaking opportunities: Encourage them to reach out to organizers\n")
	if len(config.Speakers) > 0 {
		sb.WriteString("- The speaker: Share bio and talk topic\n")
	}

	// Encouragement messages
	if len(config.BotInstructions.Encourage) > 0 {
		sb.WriteString("\nBe encouraging:\n")
		for _, msg := range config.BotInstructions.Encourage {
			sb.WriteString(fmt.Sprintf("- %s\n", msg))
		}
	}

	sb.WriteString("\nBe welcoming to newcomers and encourage participation!")

	return sb.String()
}
