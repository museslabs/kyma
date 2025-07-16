package markdown

import (
	"bytes"
	"strings"
)

type ImageParser struct{}

func NewImageParser() *ImageParser {
	return &ImageParser{}
}

func (p ImageParser) Trigger() []byte {
	return []byte{'!'}
}

func (p ImageParser) Parse(r *bytes.Reader) Node {
	var altText bytes.Buffer

	altByte, err := r.ReadByte()
	if err != nil || altByte != '[' {
		return nil
	}

	for altByte != ']' {
		altByte, err = r.ReadByte()
		if err != nil {
			return nil
		}
		altText.WriteByte(altByte)
	}

	var path bytes.Buffer

	pathByte, err := r.ReadByte()
	if err != nil || pathByte != '(' {
		return nil
	}

	for pathByte != ')' {
		pathByte, err = r.ReadByte()
		if err != nil {
			return nil
		}
		path.WriteByte(pathByte)
	}

	return &ImageNode{
		Label: strings.Trim(altText.String(), "[]"),
		Path:  strings.Trim(path.String(), "()"),
	}
}
