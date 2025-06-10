package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
	chromaStyles "github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/glamour/ansi"
	glamourStyles "github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-viper/mapstructure/v2"
	"github.com/goccy/go-yaml"

	"github.com/museslabs/kyma/internal/tui/transitions"
)

const (
	DefaultBorderColor = "#9999CC"
	chromaStyleTheme   = "kyma"
)

var chromaMutex = sync.Mutex{}

type Properties struct {
	Title      string                 `yaml:"title"`
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
}

func (s *StyleConfig) DecodeMap(input map[string]any) error {
	aux := struct {
		Layout      string `mapstructure:"layout"`
		Border      string `mapstructure:"border"`
		BorderColor string `mapstructure:"border_color"`
		Theme       string `mapstructure:"theme"`
	}{}

	if err := mapstructure.Decode(input, &aux); err != nil {
		return err
	}

	var err error
	s.Layout, err = getLayout(aux.Layout)
	if err != nil {
		return err
	}
	s.Border = getBorder(aux.Border)
	s.BorderColor = aux.BorderColor
	s.Theme = getTheme(aux.Theme)

	return nil
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

func (s StyleConfig) Apply(width, height int) SlideStyle {
	borderColor := DefaultBorderColor

	if s.Theme.Style.H1.BackgroundColor != nil {
		borderColor = *s.Theme.Style.H1.BackgroundColor
	}

	if s.BorderColor != "" {
		borderColor = s.BorderColor
	}

	if s.BorderColor == "default" {
		borderColor = DefaultBorderColor
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
		return lipgloss.NormalBorder()
	default:
		return lipgloss.Border{}
	}
}

// getLayout parses a layout string and returns a lipgloss.Style with the specified alignment.
// The layout string may contain one or two comma-separated position values (e.g., "center", "top,left").
// Returns an error if the layout string is invalid or contains more than two positions.
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

// getTheme returns a GlamourTheme for the given theme name, loading from default styles or from a JSON file if not found.
// If the theme is not recognized and cannot be loaded from file, it falls back to the default dark theme.
func getTheme(theme string) GlamourTheme {
	style, ok := glamourStyles.DefaultStyles[theme]
	if !ok {
		jsonBytes, err := os.ReadFile(theme)
		if err != nil {
			return GlamourTheme{Style: glamourStyles.DarkStyleConfig, Name: "dark"}
		}

		var customStyle ansi.StyleConfig
		if err := json.Unmarshal(jsonBytes, &customStyle); err != nil {
			return GlamourTheme{Style: glamourStyles.DarkStyleConfig, Name: "dark"}
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

func (p *Properties) UnmarshalYAML(bytes []byte) error {
	aux := struct {
		Title      string      `yaml:"title"`
		Style      StyleConfig `yaml:"style"`
		Transition string      `yaml:"transition"`
		Preset     string      `yaml:"preset"`
	}{}

	if err := yaml.Unmarshal(bytes, &aux); err != nil {
		return err
	}

	p.Title = aux.Title

	if aux.Preset != "" {
		preset, ok := GlobalConfig.Presets[aux.Preset]
		if !ok {
			return fmt.Errorf("preset %s does not exist", aux.Preset)
		}
		p.Style = preset.Style
		p.Transition = preset.Transition
	} else {
		p.Style = aux.Style
		p.Transition = transitions.Get(aux.Transition, transitions.Fps)
	}

	if p.Style.Layout.GetAlignHorizontal() == lipgloss.Left ||
		p.Style.Layout.GetAlignVertical() == lipgloss.Top { // The default
		p.Style.Layout = GlobalConfig.Global.Style.Layout
	}
	if p.Style.Border == (lipgloss.Border{}) {
		p.Style.Border = GlobalConfig.Global.Style.Border
	}
	if p.Style.BorderColor == "" {
		p.Style.BorderColor = GlobalConfig.Global.Style.BorderColor
	}
	if p.Style.Theme.Name == "" {
		p.Style.Theme = GlobalConfig.Global.Style.Theme
	}
	if p.Transition == nil {
		p.Transition = GlobalConfig.Global.Transition
	}

	return nil
}

// NewProperties creates a Properties instance from a YAML string, applying global defaults if the input is empty.
// Returns the parsed Properties and any error encountered during YAML unmarshaling.
func NewProperties(properties string) (Properties, error) {
	if properties == "" {
		return Properties{
			Style:      GlobalConfig.Global.Style,
			Transition: GlobalConfig.Global.Transition,
		}, nil
	}

	var p Properties
	if err := yaml.Unmarshal([]byte(properties), &p); err != nil {
		return Properties{}, err
	}

	return p, nil
}

// ChromaStyle converts an ansi.StylePrimitive to a Chroma-compatible style string, combining color, background color, and style attributes.
func ChromaStyle(style ansi.StylePrimitive) string {
	var s string

	if style.Color != nil {
		s = *style.Color
	}
	if style.BackgroundColor != nil {
		if s != "" {
			s += " "
		}
		s += "bg:" + *style.BackgroundColor
	}
	if style.Italic != nil && *style.Italic {
		if s != "" {
			s += " "
		}
		s += "italic"
	}
	if style.Bold != nil && *style.Bold {
		if s != "" {
			s += " "
		}
		s += "bold"
	}
	if style.Underline != nil && *style.Underline {
		if s != "" {
			s += " "
		}
		s += "underline"
	}

	return s
}

// GetChromaStyle returns a Chroma syntax highlighting style for the given theme name.
// If a custom Chroma style matching the theme exists, it is returned or created from the theme's code block style configuration.
// If no custom style is available, a predefined Chroma style is returned for known theme names, or the fallback style is used.
func GetChromaStyle(themeName string) *chroma.Style {
	customThemeName := chromaStyleTheme + "-" + themeName

	chromaMutex.Lock()
	defer chromaMutex.Unlock()

	if style, ok := chromaStyles.Registry[customThemeName]; ok {
		return style
	}

	styleConfig := getTheme(themeName)
	style := styleConfig.Style
	if style.CodeBlock.Chroma != nil {
		style := chroma.MustNewStyle(customThemeName,
			chroma.StyleEntries{
				chroma.Text:                ChromaStyle(style.CodeBlock.Chroma.Text),
				chroma.Error:               ChromaStyle(style.CodeBlock.Chroma.Error),
				chroma.Comment:             ChromaStyle(style.CodeBlock.Chroma.Comment),
				chroma.CommentPreproc:      ChromaStyle(style.CodeBlock.Chroma.CommentPreproc),
				chroma.Keyword:             ChromaStyle(style.CodeBlock.Chroma.Keyword),
				chroma.KeywordReserved:     ChromaStyle(style.CodeBlock.Chroma.KeywordReserved),
				chroma.KeywordNamespace:    ChromaStyle(style.CodeBlock.Chroma.KeywordNamespace),
				chroma.KeywordType:         ChromaStyle(style.CodeBlock.Chroma.KeywordType),
				chroma.Operator:            ChromaStyle(style.CodeBlock.Chroma.Operator),
				chroma.Punctuation:         ChromaStyle(style.CodeBlock.Chroma.Punctuation),
				chroma.Name:                ChromaStyle(style.CodeBlock.Chroma.Name),
				chroma.NameBuiltin:         ChromaStyle(style.CodeBlock.Chroma.NameBuiltin),
				chroma.NameTag:             ChromaStyle(style.CodeBlock.Chroma.NameTag),
				chroma.NameAttribute:       ChromaStyle(style.CodeBlock.Chroma.NameAttribute),
				chroma.NameClass:           ChromaStyle(style.CodeBlock.Chroma.NameClass),
				chroma.NameConstant:        ChromaStyle(style.CodeBlock.Chroma.NameConstant),
				chroma.NameDecorator:       ChromaStyle(style.CodeBlock.Chroma.NameDecorator),
				chroma.NameException:       ChromaStyle(style.CodeBlock.Chroma.NameException),
				chroma.NameFunction:        ChromaStyle(style.CodeBlock.Chroma.NameFunction),
				chroma.NameOther:           ChromaStyle(style.CodeBlock.Chroma.NameOther),
				chroma.Literal:             ChromaStyle(style.CodeBlock.Chroma.Literal),
				chroma.LiteralNumber:       ChromaStyle(style.CodeBlock.Chroma.LiteralNumber),
				chroma.LiteralDate:         ChromaStyle(style.CodeBlock.Chroma.LiteralDate),
				chroma.LiteralString:       ChromaStyle(style.CodeBlock.Chroma.LiteralString),
				chroma.LiteralStringEscape: ChromaStyle(style.CodeBlock.Chroma.LiteralStringEscape),
				chroma.GenericDeleted:      ChromaStyle(style.CodeBlock.Chroma.GenericDeleted),
				chroma.GenericEmph:         ChromaStyle(style.CodeBlock.Chroma.GenericEmph),
				chroma.GenericInserted:     ChromaStyle(style.CodeBlock.Chroma.GenericInserted),
				chroma.GenericStrong:       ChromaStyle(style.CodeBlock.Chroma.GenericStrong),
				chroma.GenericSubheading:   ChromaStyle(style.CodeBlock.Chroma.GenericSubheading),
				chroma.Background:          ChromaStyle(style.CodeBlock.Chroma.Background),
			})
		chromaStyles.Register(style)
		return style
	}

	switch themeName {
	case "dracula":
		return chromaStyles.Get("dracula")
	case "dark":
		return chromaStyles.Get("github-dark")
	case "light":
		return chromaStyles.Get("github")
	case "tokyo-night", "tokyonight":
		return chromaStyles.Get("tokyo-night")
	default:
		return chromaStyles.Fallback
	}
}
