package markdown

import (
	"bytes"
	"io"
	"log/slog"
	"slices"
	"strconv"
	"strings"
)

// layoutTags are the container names the layout parsers recognise. Closing tags
// are only honoured for these names, so markdown such as [/etc] stays literal.
// [slot] is not one of them: slots are resolved into the slide before it is
// ever parsed, see [ApplyLayout].
var layoutTags = []string{"grid", "row", "col", "column"}

// Attrs holds the attributes of a layout tag, keyed by their lowercased name.
type Attrs map[string]string

// Get returns the value of key, or def when the attribute is missing or empty.
func (a Attrs) Get(key, def string) string {
	if v, ok := a[key]; ok && v != "" {
		return v
	}
	return def
}

// Int returns the value of key parsed as an integer, or def when the attribute
// is missing or is not a valid number.
func (a Attrs) Int(key string, def int) int {
	v, ok := a[key]
	if !ok || v == "" {
		return def
	}

	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		slog.Warn(
			"invalid layout attribute",
			slog.String("attr", key),
			slog.String("value", v),
		)
		return def
	}

	return n
}

// openTag scans an opening layout tag such as [col span=2] at the reader's
// current position, where the '[' trigger byte has already been consumed. An
// opening tag is only recognised when it opens its line, which is what keeps it
// from colliding with markdown link syntax mid-sentence. The reader is left
// untouched unless the tag matches one of names.
func openTag(r *bytes.Reader, names ...string) (Attrs, bool) {
	start, _ := r.Seek(0, io.SeekCurrent)

	if !atLineStart(r) {
		return nil, false
	}

	name, attrs, ok := scanTag(r)
	if !ok || !slices.Contains(names, name) {
		_, _ = r.Seek(start, io.SeekStart)
		return nil, false
	}

	return attrs, true
}

// closeTag scans a closing layout tag such as [/col] at the reader's current
// position, where the '[' trigger byte has already been consumed, and returns
// the name it closes. Unlike opening tags a closing tag may sit at the end of a
// content line, so that [row]slave[/row] works. The reader is left untouched
// unless a tag matches.
func closeTag(r *bytes.Reader) (string, bool) {
	start, _ := r.Seek(0, io.SeekCurrent)

	rewind := func() (string, bool) {
		_, _ = r.Seek(start, io.SeekStart)
		return "", false
	}

	if b, err := r.ReadByte(); err != nil || b != '/' {
		return rewind()
	}

	var name bytes.Buffer
	for {
		b, err := r.ReadByte()
		if err != nil || b == '\n' {
			return rewind()
		}
		if b == ']' {
			break
		}
		name.WriteByte(b)
	}

	tag := strings.ToLower(strings.TrimSpace(name.String()))
	if !slices.Contains(layoutTags, tag) {
		return rewind()
	}

	// Swallow the line break when the tag ends the line, so the next opening
	// tag still starts one.
	consumeLineEnd(r)

	return tag, true
}

// scanTag reads the body of a tag up to its ']' and splits it into a lowercased
// name and its key=value attributes. When the tag ends its line the line break
// is consumed, so that content does not start with a stray blank line.
func scanTag(r *bytes.Reader) (name string, attrs Attrs, ok bool) {
	var body bytes.Buffer
	for {
		b, err := r.ReadByte()
		if err != nil || b == '\n' {
			return "", nil, false
		}
		if b == ']' {
			break
		}
		body.WriteByte(b)
	}

	consumeLineEnd(r)

	fields := splitFields(body.String())
	if len(fields) == 0 {
		return "", nil, false
	}

	name = strings.ToLower(fields[0])
	attrs = Attrs{}

	for _, f := range fields[1:] {
		k, v, isPair := strings.Cut(f, "=")
		if !isPair {
			slog.Warn(
				"ignoring layout attribute with no value",
				slog.String("tag", name),
				slog.String("attr", f),
			)
			continue
		}
		attrs[strings.ToLower(strings.TrimSpace(k))] = unquote(v)
	}

	return name, attrs, true
}

// splitFields splits a tag body on whitespace, keeping quoted values such as
// pad="1 2" in a single field.
func splitFields(s string) []string {
	var (
		fields []string
		field  strings.Builder
		quote  byte
	)

	flush := func() {
		if field.Len() > 0 {
			fields = append(fields, field.String())
			field.Reset()
		}
	}

	for i := range len(s) {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
			field.WriteByte(c)
		case c == '"' || c == '\'':
			quote = c
			field.WriteByte(c)
		case c == ' ' || c == '\t':
			flush()
		default:
			field.WriteByte(c)
		}
	}
	flush()

	return fields
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

// atLineStart reports whether the byte just consumed opens its line, meaning it
// is preceded only by whitespace or by another tag that just closed. The reader
// position is preserved.
func atLineStart(r *bytes.Reader) bool {
	pos, _ := r.Seek(0, io.SeekCurrent)
	defer func() { _, _ = r.Seek(pos, io.SeekStart) }()

	// pos-1 holds the byte that was just consumed, so scan back from pos-2.
	for i := pos - 2; i >= 0; i-- {
		if _, err := r.Seek(i, io.SeekStart); err != nil {
			return false
		}

		b, err := r.ReadByte()
		if err != nil {
			return false
		}
		// A ']' here ended the previous tag, so [grid][col] reads as two tags
		// while a link in the middle of a sentence still does not.
		if b == '\n' || b == ']' {
			return true
		}
		if b != ' ' && b != '\t' {
			return false
		}
	}

	return true
}

// scanFence copies a fenced code block verbatim, so that layout tags written
// inside one are documented rather than laid out. The opening backtick has
// already been consumed, and the reader is left untouched when what follows is
// not a fence.
func scanFence(r *bytes.Reader) (string, bool) {
	start, _ := r.Seek(0, io.SeekCurrent)

	var out bytes.Buffer
	for range 2 {
		b, err := r.ReadByte()
		if err != nil || b != '`' {
			_, _ = r.Seek(start, io.SeekStart)
			return "", false
		}
		out.WriteByte(b)
	}

	var (
		ticks    int
		lineHead bool
	)

	for {
		b, err := r.ReadByte()
		if err != nil {
			// An unterminated fence takes the rest of the input, which is what
			// a markdown renderer does with one too.
			return out.String(), true
		}
		out.WriteByte(b)

		if b == '`' && (lineHead || ticks > 0) {
			if ticks++; ticks == 3 {
				break
			}
			continue
		}

		lineHead, ticks = b == '\n', 0
	}

	// Take the rest of the closing fence's line with it.
	for {
		b, err := r.ReadByte()
		if err != nil {
			break
		}
		out.WriteByte(b)
		if b == '\n' {
			break
		}
	}

	return out.String(), true
}

// consumeLineEnd consumes the rest of the current line together with its line
// break, reporting whether that remainder was blank. The reader is left where
// it started when it was not.
func consumeLineEnd(r *bytes.Reader) bool {
	start, _ := r.Seek(0, io.SeekCurrent)

	for {
		b, err := r.ReadByte()
		if err != nil {
			return true
		}
		if b == '\n' {
			return true
		}
		if b != ' ' && b != '\t' && b != '\r' {
			_, _ = r.Seek(start, io.SeekStart)
			return false
		}
	}
}
