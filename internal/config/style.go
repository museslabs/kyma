package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

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

type Properties struct {
	Title        string                 `yaml:"title"`
	Style        StyleConfig            `yaml:"style"`
	Transition   transitions.Transition `yaml:"transition"`
	Notes        string                 `yaml:"notes"`
	ImageBackend string                 `yaml:"image_backend"`
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
	Layout      *lipgloss.Style  `yaml:"layout"`
	Border      *lipgloss.Border `yaml:"border"`
	BorderColor string           `yaml:"border_color"`
	Theme       *GlamourTheme    `yaml:"theme"`
}

func (s *StyleConfig) Merge(style StyleConfig) {
	if style.Layout != nil {
		s.Layout = style.Layout
	}
	if style.Border != nil {
		s.Border = style.Border
	}
	if style.BorderColor != "" {
		s.BorderColor = style.BorderColor
	}
	if style.Theme != nil {
		s.Theme = style.Theme
	}
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

	layout, err, ok := getLayout(aux.Layout)
	if err != nil {
		return err
	}
	if ok {
		s.Layout = &layout
	}

	if border, ok := getBorder(aux.Border); ok {
		s.Border = &border
	}

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

	layout, err, ok := getLayout(aux.Layout)
	if err != nil {
		return err
	}
	if ok {
		s.Layout = &layout
	}

	if border, ok := getBorder(aux.Border); ok {
		s.Border = &border
	}

	s.BorderColor = aux.BorderColor
	s.Theme = getTheme(aux.Theme)

	return nil
}

func (s StyleConfig) Apply(width, height int) SlideStyle {
	borderColor := DefaultBorderColor

	theme := GlamourTheme{Style: glamourStyles.DarkStyleConfig, Name: "dark"}
	if s.Theme != nil {
		theme = *s.Theme
	}

	if theme.Style.H1.BackgroundColor != nil {
		borderColor = *theme.Style.H1.BackgroundColor
	}

	if s.BorderColor != "" {
		borderColor = s.BorderColor
	}

	if s.BorderColor == "default" {
		borderColor = DefaultBorderColor
	}

	layout := lipgloss.NewStyle()
	if s.Layout != nil {
		layout = *s.Layout
	}

	border := lipgloss.RoundedBorder()
	if s.Border != nil {
		border = *s.Border
	}

	style := layout.
		Border(border).
		BorderForeground(lipgloss.Color(borderColor)).
		Width(width - 2).
		Height(height - 2)

	return SlideStyle{
		LipGlossStyle: style,
		Theme:         theme,
	}
}

func getBorder(border string) (lipgloss.Border, bool) {
	switch border {
	case "rounded":
		return lipgloss.RoundedBorder(), true
	case "double":
		return lipgloss.DoubleBorder(), true
	case "thick":
		return lipgloss.ThickBorder(), true
	case "hidden":
		return lipgloss.HiddenBorder(), true
	case "block":
		return lipgloss.BlockBorder(), true
	case "innerHalfBlock":
		return lipgloss.InnerHalfBlockBorder(), true
	case "outerHalfBlock":
		return lipgloss.OuterHalfBlockBorder(), true
	case "normal":
		return lipgloss.NormalBorder(), true
	default:
		return lipgloss.Border{}, false
	}
}

func getLayout(layout string) (lipgloss.Style, error, bool) {
	layout = strings.TrimSpace(layout)
	if layout == "" {
		return lipgloss.Style{}, nil, false
	}

	positions := strings.Split(layout, ",")
	if len(positions) > 2 {
		return lipgloss.Style{}, fmt.Errorf("invalid layout configuration: %s", layout), false
	}

	p1, err := getLayoutPosition(positions[0])
	if err != nil {
		return lipgloss.Style{}, err, false
	}

	if len(positions) == 1 {
		return lipgloss.NewStyle().Align(p1, p1), nil, true
	}

	p2, err := getLayoutPosition(positions[1])
	if err != nil {
		return lipgloss.Style{}, err, false
	}

	return lipgloss.NewStyle().Align(p1, p2), nil, true
}

func getTheme(theme string) *GlamourTheme {
	style, ok := glamourStyles.DefaultStyles[theme]
	if !ok {
		jsonBytes, err := os.ReadFile(theme)
		if err != nil {
			return nil
		}

		var customStyle ansi.StyleConfig
		if err := json.Unmarshal(jsonBytes, &customStyle); err != nil {
			return nil
		}

		return &GlamourTheme{Style: customStyle, Name: theme}
	}

	return &GlamourTheme{Style: *style, Name: theme}
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
		Title        string      `yaml:"title"`
		Style        StyleConfig `yaml:"style"`
		Transition   string      `yaml:"transition"`
		Preset       string      `yaml:"preset"`
		Notes        string      `yaml:"notes"`
		ImageBackend string      `yaml:"image_backend"`
	}{}

	if err := aux.Style.UnmarshalYAML(bytes); err != nil {
		return err
	}
	if err := yaml.Unmarshal(bytes, &aux); err != nil {
		return err
	}

	p.Title = aux.Title
	p.Notes = aux.Notes
	p.ImageBackend = aux.ImageBackend

	if aux.Preset != "" {
		preset, ok := GlobalConfig.Presets[aux.Preset]
		if !ok {
			return fmt.Errorf("preset %s does not exist", aux.Preset)
		}
		preset.Style.Merge(aux.Style)
		p.Style = preset.Style
		p.Transition = preset.Transition
	} else {
		style := GlobalConfig.Global.Style
		style.Merge(aux.Style)
		p.Style = style
		p.Transition = transitions.Get(aux.Transition, transitions.Fps)
	}

	if p.Transition == nil {
		p.Transition = GlobalConfig.Global.Transition
	}
	if p.ImageBackend == "" {
		p.ImageBackend = "chafa"
	}

	return nil
}

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

func ChromaStyle(style ansi.StylePrimitive) string {
	var s string

	if style.Color != nil && *style.Color != "" {
		s = *style.Color
	}
	if style.BackgroundColor != nil && *style.BackgroundColor != "" {
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

	if s == "" {
		return "inherit"
	}

	return s
}

func GetChromaStyle(themeName string) *chroma.Style {
	customThemeName := chromaStyleTheme + "-" + themeName

	if chromaStyle, ok := chromaStyles.Registry[customThemeName]; ok {
		return chromaStyle
	}

	styleConfig := GlamourTheme{Style: glamourStyles.DarkStyleConfig, Name: "dark"}
	if theme := getTheme(themeName); theme != nil {
		styleConfig = *theme
	}

	style := styleConfig.Style

	if style.CodeBlock.Chroma != nil {
		newStyle := chroma.MustNewStyle(customThemeName,
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

		if existingStyle, ok := chromaStyles.Registry[customThemeName]; ok {
			return existingStyle
		}
		chromaStyles.Register(newStyle)
		return newStyle
	}

	switch themeName {
	case "dracula":
		return chromaStyles.Get("dracula")
	case "tokyo-night", "tokyonight":
		return chromaStyles.Get("tokyo-night")
	default:
		return chromaStyles.Fallback
	}
}
