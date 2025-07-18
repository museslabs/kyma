package markdown

import (
	"bytes"
	"errors"
	"fmt"
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

func (p *CodeBlockParser) Parse(r *bytes.Reader) Node {
	for range 2 {
		if b, err := r.ReadByte(); err != nil || b != '`' {
			return nil
		}
	}

	var language bytes.Buffer
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil
		}
		if b == '\n' {
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
		return nil
	}

	flags, err := p.parseFlags(r)
	if err != nil {
		return nil
	}

	var code bytes.Buffer
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil
		}
		if b == '`' {
			break
		}
		code.WriteByte(b)
	}

	for range 2 {
		if b, err := r.ReadByte(); err != nil || b != '`' {
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
