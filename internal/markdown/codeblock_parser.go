package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

var (
	ErrLineRangeEnd            = errors.New("finished parsing line range")
	ErrStartLineGreaterThanEnd = errors.New("start line is greater than end line")
)

type CodeBlockParser struct{}

func NewCodeBlockParser() *CodeBlockParser {
	return &CodeBlockParser{}
}

func (p CodeBlockParser) Trigger() []byte {
	return []byte{'`'}
}

// Parse extracts a [CodeBlockNode] from the input, matching extended markdown
// code block syntax. Supports language tags, line highlighting (e.g., {1-4}),
// and custom flags (e.g., --numbered):
//
//	```c{1-4} --numbered
//	int main(void) {
//	  return 0;
//	}
//	```
func (p *CodeBlockParser) Parse(r *bytes.Reader) Node {
	for range 2 {
		if b, err := r.ReadByte(); err != nil || b != '`' {
			slog.Warn("failed to parse codeblock node")
			return nil
		}
	}

	var language bytes.Buffer
	for {
		b, err := r.ReadByte()
		if err != nil {
			slog.Warn("failed to advance reader", slog.Any("error", err))
			return nil
		}
		if b == '\n' {
			slog.Warn("failed to parse codeblock node")
			return nil
		}

		if b == ' ' {
			continue
		}

		if b == '{' {
			break
		}

		language.WriteByte(b)
	}

	lines, err := p.parseLines(r)
	if err != nil {
		slog.Warn("failed to parse codeblock lines", slog.Any("error", err))
		return nil
	}

	flags, err := p.parseFlags(r)
	if err != nil {
		slog.Warn("failed to parse codeblock flags", slog.Any("error", err))
		return nil
	}

	var code bytes.Buffer
	for {
		b, err := r.ReadByte()
		if err != nil {
			slog.Warn("failed to advance reader", slog.Any("error", err))
			return nil
		}
		if b == '`' {
			break
		}
		code.WriteByte(b)
	}

	for range 2 {
		if b, err := r.ReadByte(); err != nil || b != '`' {
			slog.Warn("failed to parse codeblock node")
			return nil
		}
	}

	return &CodeBlockNode{
		Language:        language.String(),
		Code:            strings.Trim(code.String(), "\n"),
		Ranges:          lines,
		ShowLineNumbers: flags.showLineNumbers,
		StartLine:       flags.startLine,
	}
}

// parseLines parses a comma-separated list of line ranges enclosed in braces,
// such as {1,4-5,6}, using [CodeBlockParser.parseLineRange] for each range.
func (p *CodeBlockParser) parseLines(r *bytes.Reader) ([]CodeBlockLineRange, error) {
	var num bytes.Buffer
	var lines []CodeBlockLineRange

	for {
		lr, err := p.parseLineRange(r, num, CodeBlockLineRange{})
		if err != nil {
			if errors.Is(err, ErrLineRangeEnd) {
				lines = append(lines, lr)
				break
			} else {
				return nil, err
			}
		}
		lines = append(lines, lr)
	}

	return lines, nil
}

// parseLineRange is a recursive function that parses a single line range segment
// from the input reader. It supports individual lines (e.g., 3) and ranges (e.g., 1-4),
// terminating when a closing brace '}' or a comma ',' is encountered.
func (p *CodeBlockParser) parseLineRange(
	r *bytes.Reader,
	num bytes.Buffer,
	lineRange CodeBlockLineRange,
) (CodeBlockLineRange, error) {
	b, err := r.ReadByte()
	if err != nil {
		return CodeBlockLineRange{}, err
	}

	switch {
	case b == '}':
		n, err := strconv.Atoi(num.String())
		if err != nil {
			return CodeBlockLineRange{}, err
		}
		if lineRange.Start != 0 {
			if n < lineRange.Start {
				return CodeBlockLineRange{}, ErrStartLineGreaterThanEnd
			}
			lineRange.End = n
			return lineRange, ErrLineRangeEnd
		}
		return CodeBlockLineRange{Start: n, End: n}, ErrLineRangeEnd

	case b == ',':
		n, err := strconv.Atoi(num.String())
		if err != nil {
			return CodeBlockLineRange{}, err
		}
		if lineRange.Start != 0 {
			if n < lineRange.Start {
				return CodeBlockLineRange{}, ErrStartLineGreaterThanEnd
			}
			lineRange.End = n
			return lineRange, nil
		}
		return CodeBlockLineRange{Start: n, End: n}, nil

	case b == '-':
		if lineRange.Start != 0 {
			return CodeBlockLineRange{}, errors.New("invalid range syntax")
		}

		n, err := strconv.Atoi(num.String())
		if err != nil {
			return CodeBlockLineRange{}, err
		}
		num.Reset()
		lineRange.Start = n
		return p.parseLineRange(r, num, lineRange)

	case b >= '1' && b <= '9':
		num.WriteByte(b)
		return p.parseLineRange(r, num, lineRange)

	case b == ' ':
		return p.parseLineRange(r, num, lineRange)

	default:
		return CodeBlockLineRange{}, errors.New("invalid character")
	}
}

type codeblockFlags struct {
	showLineNumbers bool
	startLine       int
}

// parseFlags parses custom [codeblockFlags].
func (p *CodeBlockParser) parseFlags(r *bytes.Reader) (codeblockFlags, error) {
	flags := codeblockFlags{showLineNumbers: false, startLine: 1}

	var flagBuf bytes.Buffer
	for {
		b, err := r.ReadByte()
		if err != nil {
			return codeblockFlags{}, err
		}
		if b == '\n' {
			break
		}
		flagBuf.WriteByte(b)
	}

	parts := strings.Fields(flagBuf.String())
	for i := 0; i < len(parts); i++ {
		switch parts[i] {
		case "--numbered":
			flags.showLineNumbers = true
		case "--start-at-line":
			if i+1 >= len(parts) {
				return flags, fmt.Errorf("missing value for --start-at-line")
			}
			n, err := strconv.Atoi(parts[i+1])
			if err != nil {
				return flags, fmt.Errorf("invalid number for --start-at-line: %v", err)
			}
			flags.startLine = n
			i++
		}
	}

	return flags, nil
}
