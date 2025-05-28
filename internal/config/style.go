package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/goccy/go-yaml"
	"github.com/museslabs/kyma/internal/tui/transitions"
)

type Properties struct {
	Style      StyleConfig            `yaml:"style"`
	Transition transitions.Transition `yaml:"transition"`
}

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
	Preset      string          `yaml:"preset"`
}

func (s *StyleConfig) Merge(other StyleConfig) {
	if other.Layout.GetAlignHorizontal() != lipgloss.Left || other.Layout.GetAlignVertical() != lipgloss.Top { // Not the default
		s.Layout = other.Layout
	}
	if other.Border != (lipgloss.Border{}) {
		s.Border = other.Border
	}
	if other.BorderColor != "" {
		s.BorderColor = other.BorderColor
	}
	if other.Theme.Name != "" {
		s.Theme = other.Theme
	}
}

func (s *StyleConfig) UnmarshalYAML(bytes []byte) error {
	aux := struct {
		Layout      string `yaml:"layout"`
		Border      string `yaml:"border"`
		BorderColor string `yaml:"border_color"`
		Theme       string `yaml:"theme"`
		Preset      string `yaml:"preset"`
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
	s.Preset = aux.Preset

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

func getTheme(theme string) GlamourTheme {
	style, ok := styles.DefaultStyles[theme]
	if !ok {
		jsonBytes, err := os.ReadFile(theme)
		if err != nil {
			return GlamourTheme{Style: styles.DarkStyleConfig, Name: "dark"}
		}

		var customStyle ansi.StyleConfig
		if err := json.Unmarshal(jsonBytes, &customStyle); err != nil {
			return GlamourTheme{Style: styles.DarkStyleConfig, Name: "dark"}
		}

		return GlamourTheme{Style: customStyle, Name: theme}
	}

	return GlamourTheme{Style: *style, Name: theme}
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

func ParseStyleConfig(layout, border, borderColor, theme, preset string) (*StyleConfig, error) {
	config := &StyleConfig{}

	var err error
	if layout != "" {
		config.Layout, err = getLayout(layout)
		if err != nil {
			return nil, fmt.Errorf("failed to parse layout: %w", err)
		}
	}

	if border != "" {
		config.Border = getBorder(border)
	}

	if borderColor != "" {
		config.BorderColor = borderColor
	}

	if theme != "" {
		config.Theme = getTheme(theme)
	}

	config.Preset = preset

	return config, nil
}
