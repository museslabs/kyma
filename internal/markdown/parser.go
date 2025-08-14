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

// MarkdownParser is an extensible parser that converts a markdown string into
// a [Node] list. Custom [Parser] implementations can be registered via
// [MarkdownParser.Register], ordered by priority, and are used during parsing
// based on trigger bytes.
type MarkdownParser struct {
	registry map[byte]PrioritizedSlice[Parser]
}

// NewMarkdownParser returns a new [MarkdownParser].
func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{
		registry: map[byte]PrioritizedSlice[Parser]{},
	}
}

// Register adds a Parser to the registry with the given priority.
// Parsers with lower priority values are tried before those with higher values.
func (p *MarkdownParser) Register(parser PrioritizedValue[Parser]) {
	for _, b := range parser.Value.Trigger() {
		p.registry[b] = append(p.registry[b], parser)
		p.registry[b].Sort()
	}
}

// Parse processes the input byte-by-byte and constructs a [Node] tree,
// starting from a root [Node]. For each byte, it checks for registered Parsers
// triggered by that byte and attempts to parse using them. If no parser succeeds,
// the byte is added to a buffer. Buffered text is eventually wrapped in a
// [GlamourNode], the default [Node] type.
func (p MarkdownParser) Parse(in []byte) Node {
	r := bytes.NewReader(in)
	return p.parse(r, &MarkdownRootNode{})
}

func (p MarkdownParser) parse(r *bytes.Reader, root Node) Node {
	var chunk bytes.Buffer

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
				root.AddChild(&GlamourNode{Text: chunk.String()})
				chunk.Reset()
			}

			root.AddChild(n)
			parsed = true
		}
		if !parsed {
			chunk.WriteByte(b)
		}
	}

	if chunk.String() != "" {
		root.AddChild(&GlamourNode{Text: chunk.String()})
	}

	return root
}
