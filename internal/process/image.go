package process

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/museslabs/kyma/internal/img"
)

type ImageProcessor struct {
	backend   img.ImageBackend
	imageData map[string]string
}

func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{
		backend:   img.Get("chafa"),
		imageData: make(map[string]string),
	}
}

func (p ImageProcessor) Pre(content string, themeName string, animating bool) (string, error) {
	// Clear all images
	if animating {
		fmt.Print("\x1b_Ga=d\x1b\\")
		fmt.Print("\x1b[0m")
	}

	// Regex to match markdown image syntax: ![alt text](path)
	imageRegex := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

	matches := imageRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return content, nil
	}

	var parts []string
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
			parts = append(parts, renderMarkdownSection(beforeText, themeName))
		}

		renderedImage, err := p.backend.Render(imagePath, 10, 10, animating)
		if err != nil {
			parts = append(parts, fmt.Sprintf("[Error rendering image: %s]", altText))
			continue
		}
		parts = append(parts, renderedImage)

		lastIndex = matchEnd
	}

	if lastIndex < len(content) {
		remainingText := content[lastIndex:]
		parts = append(parts, renderMarkdownSection(remainingText, themeName))
	}

	return lipgloss.NewStyle().Width(10).Height(10).Render(strings.Join(parts, "")), nil
}

func (p ImageProcessor) Post(content string) (string, error) {
	// for token, imgData := range p.imageData {
	// 	if strings.Contains(content, token) {
	// 		content = strings.ReplaceAll(content, token, imgData)
	// 	}
	// }
	return content, nil
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
