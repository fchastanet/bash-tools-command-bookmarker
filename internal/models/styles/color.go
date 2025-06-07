package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui/colors"
)

// ColorTheme defines a collection of related colors used by the application
type ColorTheme struct {
	// Log levels
	DebugLogLevel         lipgloss.AdaptiveColor
	InfoLogLevel          lipgloss.AdaptiveColor
	ErrorLogLevel         lipgloss.AdaptiveColor
	WarnLogLevel          lipgloss.AdaptiveColor
	LogRecordAttributeKey lipgloss.AdaptiveColor

	// Help colors
	HelpKey  lipgloss.AdaptiveColor
	HelpDesc lipgloss.AdaptiveColor

	// Border colors
	InactivePreviewBorder lipgloss.AdaptiveColor
	ActivePreviewBorder   lipgloss.AdaptiveColor

	// Other UI elements
	TitleColor                 lipgloss.AdaptiveColor
	GroupReportBackgroundColor lipgloss.Color
	TaskSummaryBackgroundColor lipgloss.Color
	ScrollPercentageBackground lipgloss.AdaptiveColor
	PrimaryColor               lipgloss.Color
}

// NewDefaultColorTheme returns a new color theme with default colors
func NewDefaultColorTheme() *ColorTheme {
	return &ColorTheme{
		DebugLogLevel: lipgloss.AdaptiveColor{
			Dark:  string(colors.Blue),
			Light: string(colors.LightBlue),
		},
		InfoLogLevel: lipgloss.AdaptiveColor{
			Dark:  string(colors.Turquoise),
			Light: string(colors.Green),
		},
		ErrorLogLevel: lipgloss.AdaptiveColor{
			Dark:  string(colors.Red),
			Light: string(colors.Red),
		},
		WarnLogLevel: lipgloss.AdaptiveColor{
			Dark:  string(colors.Yellow),
			Light: string(colors.Yellow),
		},
		LogRecordAttributeKey: lipgloss.AdaptiveColor{
			Dark:  string(colors.LightGrey),
			Light: string(colors.LightGrey),
		},

		HelpKey: lipgloss.AdaptiveColor{
			Dark:  "99",
			Light: "240",
		},

		HelpDesc: lipgloss.AdaptiveColor{
			Dark:  "248",
			Light: "244",
		},

		InactivePreviewBorder: lipgloss.AdaptiveColor{
			Dark:  "238",
			Light: "250",
		},
		ActivePreviewBorder: lipgloss.AdaptiveColor{
			Dark:  string(colors.Blue),
			Light: string(colors.Cyan),
		},

		TitleColor: lipgloss.AdaptiveColor{
			Dark:  "15",
			Light: "8",
		},

		GroupReportBackgroundColor: colors.EvenLighterGrey,
		TaskSummaryBackgroundColor: colors.EvenLighterGrey,

		ScrollPercentageBackground: lipgloss.AdaptiveColor{
			Dark:  "238",
			Light: "250",
		},

		PrimaryColor: colors.Blue,
	}
}
