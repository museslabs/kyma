package markdown

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

// SizeKind describes how a grid track claims space along its axis.
type SizeKind uint8

const (
	// SizeAuto lets the track take its natural size. Columns have no natural
	// width, so a column left on SizeAuto is laid out as a single fraction and
	// ends up sharing its row evenly with its siblings.
	SizeAuto SizeKind = iota
	// SizeCells claims an exact number of terminal cells.
	SizeCells
	// SizePercent claims a percentage of the space available on the axis.
	SizePercent
	// SizeFraction claims a proportional share of the space left over once the
	// exact and natural tracks have been served.
	SizeFraction
)

// Size is the space a row or column claims along its axis.
type Size struct {
	Kind  SizeKind
	Value float64
}

func (s Size) String() string {
	switch s.Kind {
	case SizeCells:
		return strconv.Itoa(int(s.Value))
	case SizePercent:
		return trimFloat(s.Value) + "%"
	case SizeFraction:
		return trimFloat(s.Value) + "fr"
	default:
		return "auto"
	}
}

func trimFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// parseSize reads a track size such as 20, 30%, 2fr or auto. It reports false
// for anything it does not understand so callers can keep their default.
func parseSize(s string) (Size, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" || s == "auto" {
		return Size{}, false
	}

	kind := SizeCells
	switch {
	case strings.HasSuffix(s, "%"):
		kind, s = SizePercent, strings.TrimSuffix(s, "%")
	case strings.HasSuffix(s, "fr"):
		kind, s = SizeFraction, strings.TrimSuffix(s, "fr")
	}

	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || v < 0 {
		slog.Warn("invalid layout size", slog.String("value", s))
		return Size{}, false
	}

	return Size{Kind: kind, Value: v}, true
}

// Align is the placement of content inside a row or column, on either axis.
type Align uint8

const (
	// AlignDefault leaves the axis at its natural placement: left horizontally
	// and top vertically.
	AlignDefault Align = iota
	AlignStart
	AlignCenter
	AlignEnd
)

func (a Align) String() string {
	switch a {
	case AlignStart:
		return "start"
	case AlignCenter:
		return "center"
	case AlignEnd:
		return "end"
	default:
		return "default"
	}
}

// parseAlign reads an alignment, accepting both the horizontal (left/right) and
// vertical (top/bottom) spellings of each position.
func parseAlign(s string) (Align, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "left", "top", "start":
		return AlignStart, true
	case "center", "centre", "middle":
		return AlignCenter, true
	case "right", "bottom", "end":
		return AlignEnd, true
	case "":
		return AlignDefault, false
	default:
		slog.Warn("invalid layout alignment", slog.String("value", s))
		return AlignDefault, false
	}
}

// Box holds the layout attributes shared by every grid container.
type Box struct {
	// Size is the space this container claims from its parent.
	Size Size
	// Align and VAlign place the container's content inside it.
	Align  Align
	VAlign Align
	// Gap is the number of cells left between children.
	Gap int
	// PadX and PadY are the cells of padding inside the container.
	PadX int
	PadY int
	// Border names a border style from the slide style vocabulary, and is empty
	// when the container is not framed.
	Border      string
	BorderColor string
}

// parseBox reads the layout attributes common to [grid], [row] and [col].
// The sizeKeys are the attribute names that set the size on this container's
// axis, tried in order before falling back to span.
func parseBox(attrs Attrs, sizeKeys ...string) Box {
	var box Box

	for _, key := range sizeKeys {
		if size, ok := parseSize(attrs.Get(key, "")); ok {
			box.Size = size
			break
		}
	}
	if box.Size.Kind == SizeAuto {
		if span := attrs.Get("span", ""); span != "" {
			if v, err := strconv.ParseFloat(span, 64); err == nil && v > 0 {
				box.Size = Size{Kind: SizeFraction, Value: v}
			} else {
				slog.Warn("invalid layout span", slog.String("value", span))
			}
		}
	}

	if a, ok := parseAlign(attrs.Get("align", "")); ok {
		box.Align = a
	}
	if a, ok := parseAlign(attrs.Get("valign", "")); ok {
		box.VAlign = a
	}

	box.Gap = max(attrs.Int("gap", 0), 0)
	box.PadX, box.PadY = parsePadding(attrs.Get("pad", ""))
	box.Border = strings.TrimSpace(attrs.Get("border", ""))
	box.BorderColor = strings.TrimSpace(attrs.Get("border_color", ""))

	return box
}

// parsePadding reads a CSS-style padding shorthand: one value pads every side,
// two values pad "vertical horizontal".
func parsePadding(s string) (x, y int) {
	fields := strings.Fields(s)
	switch len(fields) {
	case 0:
		return 0, 0
	case 1:
		n, err := strconv.Atoi(fields[0])
		if err != nil || n < 0 {
			slog.Warn("invalid layout padding", slog.String("value", s))
			return 0, 0
		}
		return n, n
	default:
		v, errV := strconv.Atoi(fields[0])
		h, errH := strconv.Atoi(fields[1])
		if errV != nil || errH != nil || v < 0 || h < 0 {
			slog.Warn("invalid layout padding", slog.String("value", s))
			return 0, 0
		}
		return h, v
	}
}

func (b Box) attrString() string {
	var parts []string

	if b.Size.Kind != SizeAuto {
		parts = append(parts, "size: "+b.Size.String())
	}
	if b.Align != AlignDefault {
		parts = append(parts, "align: "+b.Align.String())
	}
	if b.VAlign != AlignDefault {
		parts = append(parts, "valign: "+b.VAlign.String())
	}
	if b.Gap != 0 {
		parts = append(parts, fmt.Sprintf("gap: %d", b.Gap))
	}
	if b.PadX != 0 || b.PadY != 0 {
		parts = append(parts, fmt.Sprintf("pad: %d %d", b.PadY, b.PadX))
	}
	if b.Border != "" {
		parts = append(parts, "border: "+b.Border)
	}
	if b.BorderColor != "" {
		parts = append(parts, "border_color: "+b.BorderColor)
	}

	return strings.Join(parts, ", ")
}

// distribute divides total cells between tracks, leaving gap cells between
// neighbours. Exact and percentage tracks are served first, then natural[i] is
// given to any auto track, and whatever is left over is split between the
// fraction tracks in proportion to their value. When the tracks ask for more
// room than there is, every track is shrunk proportionally so the result always
// adds up to total.
func distribute(total, gap int, sizes []Size, natural []int) []int {
	if len(sizes) == 0 {
		return nil
	}

	out := make([]int, len(sizes))

	avail := total - gap*(len(sizes)-1)
	if avail <= 0 {
		return out
	}

	var (
		used      int
		fractions float64
	)

	for i, s := range sizes {
		switch s.Kind {
		case SizeCells:
			out[i] = int(s.Value)
		case SizePercent:
			out[i] = int(s.Value / 100 * float64(avail))
		case SizeFraction:
			fractions += s.Value
			continue
		default:
			if i < len(natural) {
				out[i] = natural[i]
			}
		}
		out[i] = max(out[i], 0)
		used += out[i]
	}

	// Hand the remainder to the fraction tracks largest-first, so a 2fr track
	// always ends up at least as wide as a 1fr one.
	if remaining := avail - used; fractions > 0 {
		remaining = max(remaining, 0)

		allocated := 0
		for i, s := range sizes {
			if s.Kind != SizeFraction {
				continue
			}
			out[i] = int(s.Value / fractions * float64(remaining))
			allocated += out[i]
		}
		for i := 0; allocated < remaining; i = (i + 1) % len(sizes) {
			if sizes[i].Kind == SizeFraction {
				out[i]++
				allocated++
			}
		}
		used += allocated
	}

	if used > avail {
		shrink(out, avail, used)
	}

	return out
}

func sum(values []int) int {
	var total int
	for _, v := range values {
		total += v
	}
	return total
}

// shrink scales tracks down proportionally until they add up to avail.
func shrink(out []int, avail, used int) {
	scaled := 0
	for i, v := range out {
		out[i] = v * avail / used
		scaled += out[i]
	}
	for i := 0; scaled < avail; i = (i + 1) % len(out) {
		out[i]++
		scaled++
	}
}
