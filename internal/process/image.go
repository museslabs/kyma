package process

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/x/ansi"

	"github.com/museslabs/kyma/internal/img"
)

type cache struct {
	l string
	h string
}

type ImageProcessor struct {
	backend img.ImageBackend
	cache   cache
}

func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{
		backend: img.Get("chafa"),
		cache:   cache{},
	}
}

func clearKittyImg() string {
	var b strings.Builder

	clearAllCmd := "\x1b_Ga=d\x1b\\"
	b.WriteString(clearAllCmd)

	clearPlacementsCmd := "\x1b_Ga=d,p=1\x1b\\"
	b.WriteString(clearPlacementsCmd)

	b.WriteString("\x1b[0m")

	return b.String()
}

func (p *ImageProcessor) Pre(content string, themeName string, animating bool) (string, error) {
	var b strings.Builder

	b.WriteString(clearKittyImg())

	// Regex to match markdown image syntax: ![alt text](path)
	imageRegex := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

	matches := imageRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		b.WriteString(renderMarkdownSection(content, themeName))
		return b.String(), nil
	}

	lastIndex := 0
	indices := imageRegex.FindAllStringIndex(content, -1)

	for i, match := range matches {
		matchStart := indices[i][0]
		matchEnd := indices[i][1]

		// fullMatch := match[0] // Full match: ![alt](path)
		altText := match[1]   // Alt text
		imagePath := match[2] // Image path

		if matchStart > lastIndex {
			beforeText := content[lastIndex:matchStart]
			b.WriteString(renderMarkdownSection(beforeText, themeName))
		}

		if p.cache.h == "" {
			himg, err := p.backend.Render(imagePath, 10, 5, true)
			if err != nil {
				b.WriteString(fmt.Sprintf("[Error rendering image: %s]", altText))
				lastIndex = matchEnd
				continue
			}
			p.cache.h = himg
		}

		if p.cache.l == "" {
			limg, err := p.backend.Render(imagePath, 10, 5, false)
			if err != nil {
				b.WriteString(fmt.Sprintf("[Error rendering image: %s]", altText))
				lastIndex = matchEnd
				continue
			}
			p.cache.l = limg
		}

		if !animating {
			b.WriteString(ansi.SaveCursor)
			b.WriteString(p.cache.l)
			b.WriteString(ansi.RestoreCursor)
			b.WriteString(p.cache.h)
		} else {
			b.WriteString(p.cache.l)
		}

		lastIndex = matchEnd
	}

	if lastIndex < len(content) {
		remainingText := content[lastIndex:]
		b.WriteString(renderMarkdownSection(remainingText, themeName))
	}

	return b.String(), nil
}

func renderMarkdownSection(text, themeName string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}

	rendered, err := glamour.Render(text, themeName)
	if err != nil {
		return text
	}
	return rendered
}
