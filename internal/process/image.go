package process

import (
	"fmt"
	"regexp"
	"strings"

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

const imgPlaceholderToken = "{{IMG-TOKEN-%d}}"

func (p ImageProcessor) Pre(content string) (string, error) {
	// Clear all images
	fmt.Print("\x1b_Ga=d\x1b\\")

	// Regex to match markdown image syntax: ![alt text](path)
	imageRegex := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

	matches := imageRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return content, nil
	}

	for i, match := range matches {
		fullMatch := match[0] // Full match: ![alt](path)
		altText := match[1]   // Alt text
		imagePath := match[2] // Image path

		token := fmt.Sprintf(imgPlaceholderToken, i)

		renderedImage, err := p.backend.Render(imagePath, 10, 10)
		if err != nil {
			p.imageData[token] = fmt.Sprintf("[Error rendering image: %s]", altText)
			content = strings.Replace(content, fullMatch, token, 1)
			continue
		}

		p.imageData[token] = renderedImage
		content = strings.Replace(content, fullMatch, token, 1)
	}

	return content, nil
}

func (p ImageProcessor) Post(content string) (string, error) {
	for token, imgData := range p.imageData {
		if strings.Contains(content, token) {
			content = strings.ReplaceAll(content, token, imgData)
		}
	}
	return content, nil
}
