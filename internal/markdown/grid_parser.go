package markdown

import (
	"bytes"
	"io"
	"log/slog"
	"strings"
)

type GridParser struct{}

func NewGridParser() *GridParser {
	return &GridParser{}
}

func (p GridParser) Trigger() []byte {
	return []byte{'['}
}

func (p GridParser) Parse(r *bytes.Reader, m *MarkdownParser) Node {
	gridNode := &GridNode{}

	if ok, err := isTag(r, "grid"); !ok || err != nil {
		return nil
	}

	if err := consumeWhitespace(r); err != nil {
		slog.Warn("failed to consume whitespace", slog.Any("error", err))
		return nil
	}

	var chunk bytes.Buffer
	for {
		b, err := r.ReadByte()
		if err != nil {
			slog.Warn("failed to advance reader", slog.Any("error", err))
			return nil
		}

		if b == '[' {
			marked, _ := r.Seek(0, io.SeekCurrent)
			ok, err := isTag(r, "/grid")
			if err != nil {
				return nil
			}
			if ok {
				if err := consumeWhitespace(r); err != nil {
					slog.Warn("failed to consume whitespace", slog.Any("error", err))
					return nil
				}
				break
			}
			_, _ = r.Seek(marked, io.SeekStart)
		}

		n := m.parseNode(r, b)
		if n == nil {
			chunk.WriteByte(b)
			continue
		}

		if c := strings.Trim(chunk.String(), " \n"); c != "" {
			gridNode.AddChild(&GlamourNode{Text: chunk.String()})
			chunk.Reset()
		}
		gridNode.AddChild(n)
	}

	if c := strings.Trim(chunk.String(), " \n"); c != "" {
		gridNode.AddChild(&GlamourNode{Text: chunk.String()})
	}

	return gridNode
}

type GridColumnParser struct{}

func NewGridColumnParser() *GridColumnParser {
	return &GridColumnParser{}
}

func (p GridColumnParser) Trigger() []byte {
	return []byte{'['}
}

func (p GridColumnParser) Parse(r *bytes.Reader, m *MarkdownParser) Node {
	columnNode := &GridColumnNode{}

	if ok, err := isTag(r, "column"); !ok || err != nil {
		return nil
	}

	if err := consumeWhitespace(r); err != nil {
		slog.Warn("failed to consume whitespace", slog.Any("error", err))
		return nil
	}

	var chunk bytes.Buffer
	for {
		b, err := r.ReadByte()
		if err != nil {
			slog.Warn("failed to advance reader", slog.Any("error", err))
			return nil
		}

		if b == '[' {
			marked, _ := r.Seek(0, io.SeekCurrent)
			ok, err := isTag(r, "/column")
			if err != nil {
				return nil
			}
			if ok {
				if err := consumeWhitespace(r); err != nil {
					slog.Warn("failed to consume whitespace", slog.Any("error", err))
					return nil
				}
				break
			}
			_, _ = r.Seek(marked, io.SeekStart)
		}

		n := m.parseNode(r, b)
		if n == nil {
			chunk.WriteByte(b)
			continue
		}

		if c := strings.Trim(chunk.String(), " \n"); c != "" {
			columnNode.AddChild(&GlamourNode{Text: c})
			chunk.Reset()
		}
		columnNode.AddChild(n)
	}

	if c := strings.Trim(chunk.String(), " \n"); c != "" {
		columnNode.AddChild(&GlamourNode{Text: c})
	}

	return columnNode
}

func isTag(r *bytes.Reader, tagName string) (bool, error) {
	var (
		tag bytes.Buffer
		b   byte
		err error
	)

	for b != ']' {
		b, err = r.ReadByte()
		if err != nil {
			slog.Warn("failed to advance reader", slog.Any("error", err))
			return false, err
		}

		tag.WriteByte(b)
	}

	return strings.Trim(tag.String(), "[]") == tagName, nil
}

func consumeWhitespace(r *bytes.Reader) error {
	for {
		b, err := r.ReadByte()
		if err != nil {
			slog.Warn("failed to advance reader", slog.Any("error", err))
			return nil
		}

		if b != '\n' && b != ' ' {
			return r.UnreadByte()
		}
	}
}
