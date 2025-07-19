package markdown

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
)

// Parser defines the interface for parsing specific [Node] types. It can be
// registered with a [MarkdownParser] to handle custom parsing logic triggered
// by specific input bytes.
type Parser interface {
	// Trigger returns a slice of bytes. If any of these bytes are encountered
	// in the input, the parser will be considered for parsing.
	Trigger() []byte

	// Parse attempts to parse a [Node] from the reader. It should return nil if
	// parsing fails, allowing [MarkdownParser.Parse] to try the next parser.
	// Note: The trigger byte has already been consumed before calling Parse.
	Parse(r *bytes.Reader) Node
}

type MarkdownParser struct {
	registry map[byte]PrioritizedSlice[Parser]
}

func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{
		registry: map[byte]PrioritizedSlice[Parser]{},
	}
}

func (p *MarkdownParser) Register(parser PrioritizedValue[Parser]) {
	for _, b := range parser.Value.Trigger() {
		p.registry[b] = append(p.registry[b], parser)
		p.registry[b].Sort()
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

		parsers, ok := p.registry[b]
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
