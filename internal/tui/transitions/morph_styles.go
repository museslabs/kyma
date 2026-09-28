package transitions

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/cellbuf"
	"github.com/lucasb-eyer/go-colorful"
)

// styles numbers the styles of a scene, so a glyph refers to its style by a
// small number. The styles the two slides use come first, each with its escape
// sequence put together once. After them come the looks worked out for the
// frame being drawn, faded or blended from those. They are dropped when the
// next frame starts, and written out as they are needed rather than kept as
// strings, since an image fading out has a look for every one of its cells.
type styles struct {
	all []cellbuf.Style
	// seqs switch to each style from any other, params are its attributes
	// without its colors, colors are its foreground, background and
	// underline colors, and shows says whether a blank shows in it.
	seqs   []string
	params []string
	colors [][3]rgb
	shows  []bool

	ids     map[cellbuf.Style]int32
	applied map[application]int32
	fixed   int

	looks  []look
	fades  [2]fadeMemo
	blends map[blend]int32
	// fg and bg are the terminal's colors for the frame being drawn.
	fg, bg rgb
}

// The colors of a style, in order.
const (
	fgColor = iota
	bgColor
	ulColor
)

// An rgb is a color as the terminal shows it, when set.
type rgb struct {
	r, g, b uint8
	set     bool
}

// A look is a style worked out for a frame: the attributes of one of the
// slides' styles, base, in colors of its own.
type look struct {
	base   int32
	colors [3]rgb
}

// A fadeMemo holds the faded look of each of the slides' styles at one
// strength, or 0 where there isn't one yet. A frame fades at two strengths at
// most, one for what leaves and one for what enters.
type fadeMemo struct {
	alpha float64
	ids   []int32
}

// A blend is a look t of the way from style a to style b.
type blend struct {
	a, b int32
	t    float64
}

// An application is an escape sequence setting the style, seq, written while
// the style was from.
type application struct {
	from int32
	seq  string
}

func newStyles() styles {
	s := styles{
		ids:     make(map[cellbuf.Style]int32),
		applied: make(map[application]int32),
		blends:  make(map[blend]int32),
	}
	// The plain style comes first, so the zero glyph is a blank in it.
	s.id(cellbuf.Style{})
	return s
}

// id returns the number of a style of the slides, numbering it if it is new.
func (s *styles) id(style cellbuf.Style) int32 {
	if id, ok := s.ids[style]; ok {
		return id
	}

	seq, params := ansi.ResetStyle, ""
	if !style.Empty() {
		// Reset and set in one sequence, so no attribute of the style
		// before carries over.
		seq = "\x1b[0;" + strings.TrimPrefix(style.Sequence(), "\x1b[")
	}
	if bare := (cellbuf.Style{Attrs: style.Attrs, UlStyle: style.UlStyle}); !bare.Empty() {
		params = strings.TrimSuffix(strings.TrimPrefix(bare.Sequence(), "\x1b["), "m")
	}

	s.all = append(s.all, style)
	s.seqs = append(s.seqs, seq)
	s.params = append(s.params, params)
	s.colors = append(s.colors, [3]rgb{toRGB(style.Fg), toRGB(style.Bg), toRGB(style.Ul)})
	s.shows = append(s.shows, style.Bg != nil ||
		style.UlStyle != cellbuf.NoUnderline ||
		style.Attrs&(cellbuf.ReverseAttr|cellbuf.StrikethroughAttr) != 0)

	id := int32(len(s.all) - 1)
	s.ids[style] = id
	return id
}

// apply returns the style that the escape sequence seq, just decoded by p,
// switches the style from to. Rendered markdown repeats a handful of sequences
// thousands of times, so most are looked up rather than parsed.
func (s *styles) apply(from int32, seq string, p *ansi.Parser) int32 {
	a := application{from, seq}
	if id, ok := s.applied[a]; ok {
		return id
	}
	pen := s.all[from]
	cellbuf.ReadStyle(p.Params(), &pen)
	id := s.id(pen)
	s.applied[a] = id
	return id
}

// fix marks the styles numbered so far as the slides' own.
func (s *styles) fix() {
	s.fixed = len(s.all)
	for i := range s.fades {
		s.fades[i] = fadeMemo{alpha: -1, ids: make([]int32, s.fixed)}
	}
}

// reset drops the looks worked out for the last frame.
func (s *styles) reset() {
	s.looks = s.looks[:0]
	clear(s.blends)
	for i := range s.fades {
		s.fades[i].alpha = -1
		clear(s.fades[i].ids)
	}
	s.fg, s.bg = toRGB(terminalFg), toRGB(terminalBg)
}

// shown reports whether a blank shows in style id.
func (s *styles) shown(id int32) bool {
	if int(id) >= s.fixed {
		id = s.looks[int(id)-s.fixed].base
	}
	return s.shows[id]
}

func (s *styles) add(l look) int32 {
	s.looks = append(s.looks, l)
	return int32(s.fixed + len(s.looks) - 1)
}

// faded returns style id at strength alpha: its colors mixed with the
// terminal's background, which it can't be told apart from at nothing. The mix
// is plain RGB, which is what showing something partly see-through over the
// background is.
func (s *styles) faded(id int32, alpha float64) int32 {
	if alpha >= 1 {
		return id
	}

	memo := s.fadeMemo(alpha)
	if v := memo[id]; v != 0 {
		return v
	}

	c := s.colors[id]
	l := look{base: id}
	l.colors[fgColor] = mix(s.bg, or(c[fgColor], s.fg), alpha)
	for _, i := range []int{bgColor, ulColor} {
		if c[i].set {
			l.colors[i] = mix(s.bg, c[i], alpha)
		}
	}

	memo[id] = s.add(l)
	return memo[id]
}

func (s *styles) fadeMemo(alpha float64) []int32 {
	for i := range s.fades {
		if s.fades[i].alpha == alpha {
			return s.fades[i].ids
		}
	}
	for i := range s.fades {
		if s.fades[i].alpha < 0 {
			s.fades[i].alpha = alpha
			return s.fades[i].ids
		}
	}
	// A third strength in one frame: start the first over.
	s.fades[0].alpha = alpha
	clear(s.fades[0].ids)
	return s.fades[0].ids
}

// blended returns the look t of the way from style a to style b. Colors blend
// in OkLab, which keeps the colors in between as bright as the ends, where
// plain RGB would pass through a muddy gray, while attributes like bold switch
// over halfway.
func (s *styles) blended(a, b int32, t float64) int32 {
	switch {
	case a == b || t <= 0:
		return a
	case t >= 1:
		return b
	}

	k := blend{a, b, t}
	if v, ok := s.blends[k]; ok {
		return v
	}

	l := look{base: a}
	if t >= 0.5 {
		l.base = b
	}
	ca, cb := s.colors[a], s.colors[b]
	for i, def := range []rgb{s.fg, s.bg} {
		if ca[i].set || cb[i].set {
			l.colors[i] = blendOkLab(or(ca[i], def), or(cb[i], def), t)
		}
	}
	l.colors[ulColor] = s.colors[l.base][ulColor]

	s.blends[k] = s.add(l)
	return s.blends[k]
}

// sgrColors introduce the foreground, background and underline colors of an
// escape sequence.
var sgrColors = [3]string{";38;2;", ";48;2;", ";58;2;"}

// appendSGR writes out the escape sequence that switches to style id from any
// other.
func (s *styles) appendSGR(out []byte, id int32) []byte {
	if int(id) < s.fixed {
		return append(out, s.seqs[id]...)
	}

	l := &s.looks[int(id)-s.fixed]
	out = append(out, "\x1b[0"...)
	if p := s.params[l.base]; p != "" {
		out = append(out, ';')
		out = append(out, p...)
	}
	for i, c := range l.colors {
		if c.set {
			out = append(out, sgrColors[i]...)
			out = strconv.AppendUint(out, uint64(c.r), 10)
			out = append(out, ';')
			out = strconv.AppendUint(out, uint64(c.g), 10)
			out = append(out, ';')
			out = strconv.AppendUint(out, uint64(c.b), 10)
		}
	}
	return append(out, 'm')
}

func toRGB(c color.Color) rgb {
	if c == nil {
		return rgb{}
	}
	r, g, b, _ := c.RGBA()
	return rgb{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), true}
}

// or is c, or def where c isn't set.
func or(c, def rgb) rgb {
	if c.set {
		return c
	}
	return def
}

// mix is alpha of the way from a to b.
func mix(a, b rgb, alpha float64) rgb {
	m := func(x, y uint8) uint8 {
		return uint8(float64(x) + (float64(y)-float64(x))*alpha + 0.5)
	}
	return rgb{m(a.r, b.r), m(a.g, b.g), m(a.b, b.b), true}
}

func blendOkLab(a, b rgb, t float64) rgb {
	c := colorful.Color{R: float64(a.r) / 255, G: float64(a.g) / 255, B: float64(a.b) / 255}
	d := colorful.Color{R: float64(b.r) / 255, G: float64(b.g) / 255, B: float64(b.b) / 255}
	r, g, bl := c.BlendOkLab(d, t).Clamped().RGB255()
	return rgb{r, g, bl, true}
}

func makeColor(c color.Color) (colorful.Color, bool) {
	if c == nil {
		return colorful.Color{}, false
	}
	return colorful.MakeColor(c)
}
