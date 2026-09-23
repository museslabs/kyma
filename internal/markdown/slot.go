package markdown

import (
	"log/slog"
	"regexp"
	"strings"
)

// DefaultSlot receives everything a slide writes outside a [slot] tag, so a
// layout that only needs one hole can name it and stay out of the way.
const DefaultSlot = "content"

var (
	slotOpenRe  = regexp.MustCompile(`^[ \t]*\[slot(?:[ \t]+([A-Za-z0-9_.-]+))?[ \t]*\]`)
	slotCloseRe = regexp.MustCompile(`\[/slot\][ \t]*$`)
	fenceRe     = regexp.MustCompile("^[ \t]*(```|~~~)")
)

// Slots holds the named blocks of content a slide fills its layout with.
type Slots map[string]string

// add appends content to a slot, so a slide may fill the same slot from more
// than one block.
func (s Slots) add(name, content string) {
	content = strings.Trim(content, " \t\n")
	if content == "" {
		return
	}

	if existing, ok := s[name]; ok {
		s[name] = existing + "\n\n" + content
		return
	}
	s[name] = content
}

// slotSpan is one piece of a document: either a stretch of plain content, or a
// slot together with whatever was written inside it.
type slotSpan struct {
	Name    string
	Content string
	IsSlot  bool
}

// ParseSlots splits a slide body into the slots it fills. Content written
// outside any [slot] tag lands in [DefaultSlot].
func ParseSlots(body string) Slots {
	slots := Slots{}

	for _, span := range splitSlots(body) {
		name := span.Name
		if !span.IsSlot {
			name = DefaultSlot
		}
		slots.add(name, span.Content)
	}

	return slots
}

// ApplyLayout fills a master layout with a slide's content, replacing each of
// the template's [slot] placeholders with the matching block from the slide. A
// placeholder the slide left unfilled keeps whatever default content the layout
// wrote inside it. An empty template returns the body with its slot tags
// removed, so a slide that names slots still renders without a layout.
func ApplyLayout(template, body string) string {
	if strings.TrimSpace(template) == "" {
		return StripSlots(body)
	}

	var (
		slots = ParseSlots(body)
		used  = map[string]bool{}
		out   []string
	)

	for _, span := range splitSlots(template) {
		if !span.IsSlot {
			out = append(out, span.Content)
			continue
		}

		used[span.Name] = true

		content := span.Content
		if filled, ok := slots[span.Name]; ok {
			content = filled
		}
		if strings.TrimSpace(content) != "" {
			out = append(out, content)
		}
	}

	for name := range slots {
		if !used[name] {
			slog.Warn("slide fills a slot the layout does not have", slog.String("slot", name))
		}
	}

	return strings.Join(out, "\n")
}

// StripSlots removes the slot tags from a body while keeping its content in
// order. It is what a slide falls back to when it names slots but has no layout
// to fill.
func StripSlots(body string) string {
	var out []string
	for _, span := range splitSlots(body) {
		if strings.TrimSpace(span.Content) != "" {
			out = append(out, span.Content)
		}
	}
	return strings.Join(out, "\n")
}

// splitSlots walks a document and returns it as a sequence of plain content and
// slots. A [slot] tag with no [/slot] before the next slot is a bare
// placeholder rather than a block, which is what lets a layout write
// [slot main] on a line of its own.
func splitSlots(doc string) []slotSpan {
	lines := strings.Split(doc, "\n")

	var (
		spans []slotSpan
		text  []string
	)

	flush := func() {
		if len(text) > 0 {
			spans = append(spans, slotSpan{Content: strings.Join(text, "\n")})
			text = nil
		}
	}

	fenced := false
	for i := 0; i < len(lines); i++ {
		// A slot tag inside a code fence is being documented, not filled.
		if fenceRe.MatchString(lines[i]) {
			fenced = !fenced
		}

		open := slotOpenRe.FindStringSubmatch(lines[i])
		if fenced || open == nil {
			text = append(text, lines[i])
			continue
		}

		name := DefaultSlot
		if open[1] != "" {
			name = strings.ToLower(open[1])
		}
		rest := lines[i][len(open[0]):]

		flush()

		// [slot name]content[/slot] on a single line.
		if closing := slotCloseRe.FindStringIndex(rest); closing != nil {
			spans = append(spans, slotSpan{Name: name, Content: rest[:closing[0]], IsSlot: true})
			continue
		}

		end := findSlotClose(lines, i+1)
		if end < 0 {
			spans = append(spans, slotSpan{Name: name, IsSlot: true})
			if strings.TrimSpace(rest) != "" {
				text = append(text, rest)
			}
			continue
		}

		body := append([]string{}, lines[i+1:end]...)
		if strings.TrimSpace(rest) != "" {
			body = append([]string{rest}, body...)
		}
		if head := lines[end][:slotCloseRe.FindStringIndex(lines[end])[0]]; strings.TrimSpace(head) != "" {
			body = append(body, head)
		}

		spans = append(spans, slotSpan{
			Name:    name,
			Content: strings.Join(body, "\n"),
			IsSlot:  true,
		})
		i = end
	}
	flush()

	return spans
}

// findSlotClose returns the line that closes the slot opened before from, or -1
// when the next thing the document does is open another slot.
func findSlotClose(lines []string, from int) int {
	for i := from; i < len(lines); i++ {
		if slotCloseRe.MatchString(lines[i]) {
			return i
		}
		if slotOpenRe.MatchString(lines[i]) {
			return -1
		}
	}
	return -1
}
