package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/goccy/go-yaml"
)

type SlideStyle struct {
	LipGlossStyle lipgloss.Style
	Theme         GlamourTheme
}
type GlamourTheme struct {
	Style ansi.StyleConfig
	Name  string
}

type StyleConfig struct {
	Layout      lipgloss.Style  `yaml:"layout"`
	Border      lipgloss.Border `yaml:"border"`
	BorderColor string          `yaml:"border_color"`
	Theme       GlamourTheme    `yaml:"theme"`
}

func (s *StyleConfig) UnmarshalYAML(bytes []byte) error {
	aux := struct {
		Layout      string `yaml:"layout"`
		Border      string `yaml:"border"`
		BorderColor string `yaml:"border_color"`
		Theme       string `yaml:"theme"`
	}{}

	var err error

	if err = yaml.Unmarshal(bytes, &aux); err != nil {
		return err
	}

	s.Layout, err = getLayout(aux.Layout)
	if err != nil {
		return err
	}

	s.Border = getBorder(aux.Border)
	s.BorderColor = aux.BorderColor
	s.Theme = getTheme(aux.Theme)

	return nil
}

func (s StyleConfig) ApplyStyle(width, height int) SlideStyle {
	defaultBorderColor := "#9999CC" // Blueish
	borderColor := defaultBorderColor

	if s.Theme.Style.H1.BackgroundColor != nil {
		borderColor = *s.Theme.Style.H1.BackgroundColor
	}

	if s.BorderColor != "" {
		borderColor = s.BorderColor
	}

	if s.BorderColor == "default" {
		borderColor = defaultBorderColor
	}

	style := s.Layout.
		Border(s.Border).
		BorderForeground(lipgloss.Color(borderColor)).
		Width(width - 4).
		Height(height - 2)

	return SlideStyle{
		LipGlossStyle: style,
		Theme:         s.Theme,
	}
}

func getBorder(border string) lipgloss.Border {
	switch border {
	case "rounded":
		return lipgloss.RoundedBorder()
	case "double":
		return lipgloss.DoubleBorder()
	case "thick":
		return lipgloss.ThickBorder()
	case "hidden":
		return lipgloss.HiddenBorder()
	case "block":
		return lipgloss.BlockBorder()
	case "innerHalfBlock":
		return lipgloss.InnerHalfBlockBorder()
	case "outerHalfBlock":
		return lipgloss.OuterHalfBlockBorder()
	case "normal":
		fallthrough
	default:
		return lipgloss.NormalBorder()
	}
}

func getLayout(layout string) (lipgloss.Style, error) {
	style := lipgloss.NewStyle()

	layout = strings.TrimSpace(layout)
	if layout == "" {
		return style, nil
	}

	positions := strings.Split(layout, ",")
	if len(positions) > 2 {
		return style, fmt.Errorf("invalid layout configuration: %s", layout)
	}

	p1, err := getLayoutPosition(positions[0])
	if err != nil {
		return style, err
	}

	if len(positions) == 1 {
		return style.Align(p1, p1), nil
	}

	p2, err := getLayoutPosition(positions[1])
	if err != nil {
		return style, err
	}

	return style.Align(p1, p2), nil
}

func getLayoutPosition(p string) (lipgloss.Position, error) {
	switch strings.TrimSpace(p) {
	case "center":
		return lipgloss.Center, nil
	case "left":
		return lipgloss.Left, nil
	case "right":
		return lipgloss.Right, nil
	case "top":
		return lipgloss.Top, nil
	case "bottom":
		return lipgloss.Bottom, nil
	default:
		return 0, fmt.Errorf("invalid position: %s", strings.TrimSpace(p))
	}
}

func getTheme(theme string) GlamourTheme {
	switch theme {
	case "ascii":
		return GlamourTheme{Style: styles.ASCIIStyleConfig, Name: "ascii"}
	case "dark":
		return GlamourTheme{Style: styles.DarkStyleConfig, Name: "dark"}
	case "dracula":
		return GlamourTheme{Style: styles.DraculaStyleConfig, Name: "dracula"}
	case "tokyo-night", "tokyonight":
		return GlamourTheme{Style: styles.TokyoNightStyleConfig, Name: "tokyo-night"}
	case "light":
		return GlamourTheme{Style: styles.LightStyleConfig, Name: "light"}
	case "notty":
		return GlamourTheme{Style: styles.NoTTYStyleConfig, Name: "notty"}
	case "pink":
		return GlamourTheme{Style: styles.PinkStyleConfig, Name: "pink"}
	default:
		return GlamourTheme{Style: styles.DarkStyleConfig, Name: "dark"}
	}
}
