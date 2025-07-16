package markdown

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
)

type Parser interface {
	Trigger() []byte
	Parse(r *bytes.Reader) Node
}

type MarkdownParser struct {
	triggers map[byte]PrioritizedSlice[Parser]
}

func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{
		triggers: map[byte]PrioritizedSlice[Parser]{},
	}
}

func (p *MarkdownParser) Register(parser PrioritizedValue[Parser]) {
	for _, b := range parser.Value.Trigger() {
		p.triggers[b] = append(p.triggers[b], parser)
		p.triggers[b].Sort()
	}
}

func (p MarkdownParser) Parse(in []byte) Node {
	r := bytes.NewReader(in)

	var (
		root  Node
		curr  Node
		chunk bytes.Buffer
	)

	for {
		b, err := r.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			slog.Error("failed advancing reader", slog.Any("error", err))
		}

		parsers, ok := p.triggers[b]
		if !ok {
			chunk.WriteByte(b)
			continue
		}

		parsed := false
		for _, parser := range parsers {
			markedPos, _ := r.Seek(0, io.SeekCurrent)
			n := parser.Value.Parse(r)
			if n == nil {
				_, _ = r.Seek(markedPos, io.SeekStart)
				continue
			}

			if chunk.String() != "" {
				if root == nil {
					root = &GlamourNode{Text: chunk.String()}
					curr = root
				} else {
					curr.SetNext(&GlamourNode{Text: chunk.String()})
					curr = curr.Next()
				}
			}

			chunk.Reset()

			if root == nil {
				root = n
				curr = root
			} else {
				curr.SetNext(n)
				curr = curr.Next()
			}

			parsed = true
		}
		if !parsed {
			chunk.WriteByte(b)
		}
	}

	if chunk.String() != "" {
		if root == nil {
			root = &GlamourNode{Text: chunk.String()}
		} else {
			curr.SetNext(&GlamourNode{Text: chunk.String()})
		}
	}

	return root
}
