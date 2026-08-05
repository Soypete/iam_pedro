package twitchirc

import (
	"testing"
)

func TestGenerateStateString(t *testing.T) {
	t.Run("generates non-empty state string", func(t *testing.T) {
		state, err := generateStateString()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(state) == 0 {
			t.Error("expected non-empty state string")
		}
	})

	t.Run("generates unique state strings", func(t *testing.T) {
		state1, _ := generateStateString()
		state2, _ := generateStateString()
		if state1 == state2 {
			t.Error("expected unique state strings")
		}
	})

	t.Run("state string is base64 encoded", func(t *testing.T) {
		state, _ := generateStateString()
		for _, c := range state {
			isValid := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '='
			if !isValid {
				t.Errorf("unexpected character %c in base64 string", c)
			}
		}
	})
}

func TestIRCStruct(t *testing.T) {
	t.Run("IRC has required fields", func(t *testing.T) {
		irc := &IRC{}
		_ = irc.logger
		_ = irc.asyncResponseCh
		_ = irc.tok
		_ = irc.oauthConf
	})
}
