package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type fakeCSI string

func (f fakeCSI) String() string { return string(f) }

func TestExtendedEngineKey(t *testing.T) {
	cases := []struct {
		name string
		msg  tea.Msg
		want string
	}{
		{"ctrl+1 csi-u", fakeCSI("?CSI[52 57 59 53 117]?"), "spotify"},
		{"ctrl+2 csi-u", fakeCSI("?CSI[53 48 59 53 117]?"), "youtube"},
		{"ctrl+1 modifyOtherKeys", fakeCSI("?CSI[50 55 59 53 59 52 57 126]?"), "spotify"},
		{"ctrl+2 modifyOtherKeys", fakeCSI("?CSI[50 55 59 53 59 53 48 126]?"), "youtube"},
		{"unrelated csi", fakeCSI("?CSI[49 59 50 65]?"), ""},
		{"plain text", fakeCSI("hello"), ""},
		{"regular key", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")}, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extendedEngineKey(tc.msg); got != tc.want {
				t.Errorf("extendedEngineKey() = %q, want %q", got, tc.want)
			}
		})
	}
}
