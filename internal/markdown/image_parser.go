package markdown

import (
	"bytes"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

type ImageParser struct{}

func NewImageParser() *ImageParser {
	return &ImageParser{}
}

func (p ImageParser) Trigger() []byte {
	return []byte{'!'}
}

// Parse attempts to extract an [ImageNode] from the input, matching the markdown
// image syntax: [alt text|widthxheight](image-path) where width and height
// values are optional.
func (p ImageParser) Parse(r *bytes.Reader) Node {
	var altText bytes.Buffer

	b, err := r.ReadByte()
	if err != nil || b != '[' {
		slog.Warn("failed to parse image node")
		return nil
	}

	for b != ']' && b != '|' {
		altText.WriteByte(b)

		b, err = r.ReadByte()
		if err != nil {
			slog.Warn("failed to advance reader", slog.Any("error", err))
			return nil
		}
	}

	width := 0
	height := 0

	if b == '|' {
		width, height, err = p.parseDimensions(r)
		if err != nil {
			slog.Warn("failed to parse image dimensions", slog.Any("error", err))
			return nil
		}
	}

	var path bytes.Buffer

	pathByte, err := r.ReadByte()
	if err != nil || pathByte != '(' {
		slog.Warn("failed to parse image node")
		return nil
	}

	for pathByte != ')' {
		path.WriteByte(pathByte)

		pathByte, err = r.ReadByte()
		if err != nil {
			slog.Warn("failed to advance reader", slog.Any("error", err))
			return nil
		}
	}

	return &ImageNode{
		Label:  strings.Trim(altText.String(), "[]"),
		Path:   strings.Trim(path.String(), "()"),
		Width:  width,
		Height: height,
	}
}

// parseDimensions parses a dimension string in the format 20x10 from the reader
// and returns the corresponding width and height.
func (p ImageParser) parseDimensions(r *bytes.Reader) (int, int, error) {
	var wbuf bytes.Buffer
	var hbuf bytes.Buffer

	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, 0, err
		}
		if b == 'x' {
			break
		} else if b >= '0' && b <= '9' {
			wbuf.WriteByte(b)
		} else {
			return 0, 0, fmt.Errorf("invalid character found %c", b)
		}
	}

	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, 0, err
		}
		if b == ']' {
			break
		} else if b >= '0' && b <= '9' {
			hbuf.WriteByte(b)
		} else {
			return 0, 0, fmt.Errorf("invalid character found %c", b)
		}
	}

	width, err := strconv.Atoi(wbuf.String())
	if err != nil {
		return 0, 0, err
	}

	height, err := strconv.Atoi(hbuf.String())
	if err != nil {
		return 0, 0, err
	}

	return width, height, nil
}
