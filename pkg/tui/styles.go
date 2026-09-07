package tui

import "github.com/charmbracelet/lipgloss"

var (
	JadeMint    = lipgloss.Color("#73daca")
	JadeGreen   = lipgloss.Color("#9ece6a")
	TokyoBlue   = lipgloss.Color("#7aa2f7")
	TokyoViolet = lipgloss.Color("#bb9af7")
	TokyoRed    = lipgloss.Color("#f7768e")
	TokyoOrange = lipgloss.Color("#ff9e64")
	TokyoYellow = lipgloss.Color("#e0af68")
	TokyoCyan   = lipgloss.Color("#2ac3de")

	BgDark       = lipgloss.Color("#16161e")
	BgCard       = lipgloss.Color("#1f2335")
	BgHighlight  = lipgloss.Color("#24283b")
	BorderDark   = lipgloss.Color("#292e42")
	BorderActive = lipgloss.Color("#414868")

	TextNormal = lipgloss.Color("#c0caf5")
	TextMuted  = lipgloss.Color("#a9b1d6")
	TextDim    = lipgloss.Color("#565f89")

	ColorSuccess = lipgloss.Color("#9ece6a")
	ColorError   = lipgloss.Color("#f7768e")
	ColorWarning = lipgloss.Color("#e0af68")

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(JadeMint)

	BadgeHifiStyle = lipgloss.NewStyle().
			Foreground(TokyoViolet).
			Background(BgHighlight).
			Padding(0, 1)

	BadgeSpotifyActive = lipgloss.NewStyle().
				Bold(true).
				Foreground(BgDark).
				Background(JadeGreen).
				Padding(0, 1)

	BadgeSpotifyInactive = lipgloss.NewStyle().
				Foreground(TextDim).
				Background(BgHighlight).
				Padding(0, 1)

	BadgeYTActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(TokyoRed).
			Padding(0, 1)

	BadgeYTInactive = lipgloss.NewStyle().
			Foreground(TextDim).
			Background(BgHighlight).
			Padding(0, 1)

	BadgePill = lipgloss.NewStyle().
			Foreground(TextMuted).
			Background(BgHighlight).
			Padding(0, 1)

	TabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(JadeMint).
			Background(BgCard).
			Padding(0, 2)

	TabInactiveStyle = lipgloss.NewStyle().
				Foreground(TextDim).
				Padding(0, 2)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderDark).
			Padding(0, 1)

	BoxActiveStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(JadeMint).
			Padding(0, 1)

	BoxYTStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(TokyoRed).
			Padding(0, 1)

	ModalBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(TokyoBlue).
			Background(BgCard).
			Padding(1, 3)

	TrackTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(TextNormal)

	ArtistStyle = lipgloss.NewStyle().
			Foreground(TextMuted)

	SubtextStyle = lipgloss.NewStyle().
			Foreground(TextDim)

	LabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(JadeMint)

	StatusSuccess = lipgloss.NewStyle().
			Foreground(JadeGreen).
			Bold(true)

	StatusFailed = lipgloss.NewStyle().
			Foreground(TokyoRed).
			Bold(true)

	StatusDownloading = lipgloss.NewStyle().
				Foreground(JadeMint).
				Bold(true)

	StatusResolving = lipgloss.NewStyle().
			Foreground(TokyoYellow).
			Bold(true)

	EqualizerStyle = lipgloss.NewStyle().
			Foreground(JadeMint).
			Bold(true)

	EqualizerSpotifyStyle = lipgloss.NewStyle().
				Foreground(JadeGreen).
				Bold(true)

	EqualizerYTStyle = lipgloss.NewStyle().
				Foreground(TokyoRed).
				Bold(true)

	HintKeyStyle = lipgloss.NewStyle().
			Foreground(TextNormal).
			Background(BgHighlight).
			Padding(0, 1)
)
