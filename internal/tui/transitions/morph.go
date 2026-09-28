package transitions

import (
	"image/color"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lucasb-eyer/go-colorful"
)

// morph carries what two slides have in common over to where the next slide
// has it, like Keynote's Magic Move: a heading that moves to the top glides
// there, a list that grows slides up to make room for the new item, a box
// moves, resizes and changes color, and whatever only one of the slides has
// fades out or in around them.
type morph struct {
	started   time.Time
	progress  float64
	animating bool
	direction direction
	scene     *scene
}

// morphDuration is how long a morph takes. It runs on the clock rather than
// counting frames, so a slow frame skips ahead instead of dragging it out, and
// it can draw as often as the screen refreshes without changing its pace.
const morphDuration = 600 * time.Millisecond

// now is the clock the morph runs on.
var now = time.Now

func newMorph(int) morph {
	return morph{}
}

func (t morph) Start(_, _ int, direction direction) Transition {
	t.animating = true
	t.progress = 0
	t.started = time.Time{}
	t.direction = direction
	t.scene = &scene{}
	return t
}

func (t morph) Animating() bool {
	return t.animating
}

func (t morph) Update() (Transition, tea.Cmd) {
	// The clock starts on the first frame rather than in Start, because the
	// frame before it renders both slides and works out the scene, which can
	// take long enough to eat into the animation.
	if t.started.IsZero() {
		t.started = now()
	}

	t.progress = float64(now().Sub(t.started)) / float64(morphDuration)
	if t.progress >= 1 {
		t.animating = false
		t.progress = 1
		return t, nil
	}

	return t, Animate(RefreshRate)
}

func (t morph) View(prev, next string) string {
	if t.scene == nil {
		s := newScene(prev, next)
		return s.frame(t.progress)
	}

	// Working out what goes where means diffing both slides, so it is done
	// once and the frames after that only draw.
	if !t.scene.built || t.scene.prev != prev || t.scene.next != next {
		*t.scene = newScene(prev, next)
	}
	return t.scene.frame(t.progress)
}

func (t morph) Name() string {
	return "morph"
}

func (t morph) Opposite() Transition {
	return newMorph(0)
}

func (t morph) Direction() direction {
	return t.direction
}

// The terminal's own colors. Text without a color of its own is drawn in
// terminalFg, and what fades in or out fades from or to terminalBg. They are
// light gray on black until SetTerminalColors says otherwise.
var (
	terminalFg = colorful.Color{R: 0.75, G: 0.75, B: 0.75}
	terminalBg = colorful.Color{}
)

// SetTerminalColors tells the morph which colors the terminal draws text and
// its background in, so fades blend into the actual background. A nil color is
// left as it was.
func SetTerminalColors(fg, bg color.Color) {
	if c, ok := makeColor(fg); ok {
		terminalFg = c
	}
	if c, ok := makeColor(bg); ok {
		terminalBg = c
	}
}

// A scene is a morph worked out between two rendered slides: the pieces and
// boxes that travel from one place to another, and those only one slide has.
type scene struct {
	prev, next    string
	built         bool
	width, height int

	styles   styles
	boxes    []boxMove
	moving   []move
	leaving  []piece
	entering []piece

	// canvas is drawn over again for every frame, so a frame allocates
	// nothing but the string it ends up as.
	canvas canvas
}

// A glyph is what a cell shows: a grapheme in one of the scene's styles. The
// zero glyph is a blank.
type glyph struct {
	text  string
	style int32
	// width is the number of cells the grapheme takes, 2 for a wide one.
	width int8
	// rest marks the cell as the right half of the wide glyph to its left.
	rest bool
}

func (g glyph) cells() int {
	return max(int(g.width), 1)
}

// A piece is a run of glyphs on one row that morphs as a unit.
type piece struct {
	x, y   int
	glyphs []glyph
	// key is what the piece looks like, its text and styles, and shape is its
	// text alone. Pieces with the same key can stand in for one another, and
	// pieces with the same shape can once their colors are blended.
	key, shape string
}

// A move is a piece both slides have, travelling from where the previous
// slide has it to where the next one does.
type move struct {
	piece
	toX, toY int
	// to is the piece as the next slide has it: the same text, and in other
	// styles when restyled is set.
	to       []glyph
	restyled bool
}

func newScene(prev, next string) scene {
	s := scene{prev: prev, next: next, built: true, styles: newStyles()}
	s.width = max(frameWidth(prev), frameWidth(next))
	s.height = max(strings.Count(prev, "\n"), strings.Count(next, "\n")) + 1

	// Boxes come off the slides first, so their edges morph as whole boxes
	// and don't stick to the text next to them.
	gridA, gridB := s.grid(prev), s.grid(next)
	framesA, framesB := s.frames(gridA), s.frames(gridB)

	// Then whole runs of text, which keeps a line together as it moves: first
	// what looks exactly the same, then what only changed color.
	runs := newMatching(s.phrases(gridA), s.phrases(gridB))
	for _, byShape := range []bool{false, true} {
		runs.byShape = byShape
		runs.inPlace()
		runs.inOrder(runs.unmatched())
		runs.nearest()
	}

	// What is left gets a second look word by word, so an edited line keeps
	// the words it didn't touch.
	wordsA, fromA := s.splitWords(runs.a, runs.ai)
	wordsB, fromB := s.splitWords(runs.b, runs.bi)
	words := newMatching(wordsA, wordsB)
	words.inPlace()
	words.inSimilarLines(fromA, fromB)

	s.moving = append(runs.moves(), words.moves()...)
	settle(s.moving)
	for i, j := range words.ai {
		if j < 0 {
			s.leaving = append(s.leaving, words.a[i])
		}
	}
	for j, i := range words.bi {
		if i < 0 {
			s.entering = append(s.entering, words.b[j])
		}
	}

	s.boxes = matchFrames(framesA, framesB, s.moving)
	s.styles.fix()
	return s
}

// grid lays a rendered slide out on glyphs the size of the scene, row by row.
func (s *scene) grid(frame string) []glyph {
	grid := make([]glyph, s.width*s.height)

	p := ansi.GetParser()
	defer ansi.PutParser(p)

	var style int32
	var state byte
	x, y := 0, 0
	for len(frame) > 0 {
		seq, width, n, newState := ansi.DecodeSequence(frame, state, p)
		state, frame = newState, frame[n:]

		switch {
		case width > 0:
			if y < s.height && x+width <= s.width {
				row := grid[y*s.width:]
				row[x] = glyph{text: seq, style: style, width: int8(width)}
				for i := 1; i < width; i++ {
					row[x+i] = glyph{rest: true}
				}
			}
			x += width
		case seq == "\n":
			x, y = 0, y+1
		case seq == "\r":
			x = 0
		case ansi.HasCsiPrefix(seq) && p.Command() == 'm':
			style = s.styles.apply(style, seq, p)
		}
	}

	return grid
}

// What only one slide has fades out over the first part of a morph and in over
// the last, the two overlapping in the middle.
const (
	fadeOutUntil = 0.6
	fadeInFrom   = 0.4
)

// frame draws the scene at progress p, from 0, the previous slide, to 1, the
// next one.
func (s *scene) frame(p float64) string {
	e := ease(min(max(p, 0), 1))
	s.styles.reset()
	s.canvas.reset(s.width, s.height)

	leaving := 1 - ramp(e, 0, fadeOutUntil)
	entering := ramp(e, fadeInFrom, 1)

	for _, b := range s.boxes {
		switch {
		case b.from != nil && b.to != nil:
			edges := b.from.edges
			if e >= 0.5 {
				edges = b.to.edges
			}
			x0, y0 := lerp(b.from.x0, b.to.x0, e), lerp(b.from.y0, b.to.y0, e)
			x1, y1 := lerp(b.from.x1, b.to.x1, e), lerp(b.from.y1, b.to.y1, e)
			style := s.styles.blended(b.from.style, b.to.style, e)
			s.canvas.box(edges, x0, y0, max(x1, x0+1), max(y1, y0+1), style, 1)
		case b.from != nil:
			f := b.from
			s.canvas.box(f.edges, f.x0, f.y0, f.x1, f.y1, s.styles.faded(f.style, leaving), leaving)
		default:
			f := b.to
			s.canvas.box(f.edges, f.x0, f.y0, f.x1, f.y1, s.styles.faded(f.style, entering), entering)
		}
	}

	for _, pc := range s.leaving {
		s.fade(pc, leaving)
	}
	for _, pc := range s.entering {
		s.fade(pc, entering)
	}

	// Moving pieces go on top, so they pass over whatever is on their way.
	for i := range s.moving {
		m := &s.moving[i]
		x, y := lerp(m.x, m.toX, e), lerp(m.y, m.toY, e)

		glyphs := m.glyphs
		if e >= 1 {
			glyphs = m.to
		}

		// A run of glyphs mostly shares one style, so the blend is only
		// looked up when the style changes.
		from, to, style := int32(-1), int32(-1), int32(0)
		for j, g := range glyphs {
			if m.restyled && e < 1 && j < len(m.to) {
				if g.style != from || m.to[j].style != to {
					from, to = g.style, m.to[j].style
					style = s.styles.blended(from, to, e)
				}
				g.style = style
			}
			s.canvas.set(x, y, g, 1)
			x += g.cells()
		}
	}

	return s.canvas.String(&s.styles)
}

// fade draws a piece that only one slide has, at strength alpha.
func (s *scene) fade(p piece, alpha float64) {
	if alpha <= 0 {
		return
	}

	from, style := int32(-1), int32(0)
	x := p.x
	for _, g := range p.glyphs {
		if g.style != from {
			from, style = g.style, s.styles.faded(g.style, alpha)
		}
		g.style = style
		s.canvas.set(x, p.y, g, float32(alpha))
		x += g.cells()
	}
}

// ease speeds a morph up out of the previous slide and slows it down into the
// next one. A sine curve does so gently: at its fastest it goes about one and a
// half times its average speed, where steeper curves make long moves skip
// cells between frames.
func ease(p float64) float64 {
	return (1 - math.Cos(math.Pi*p)) / 2
}

// ramp is how far p is from lo to hi, from 0 to 1.
func ramp(p, lo, hi float64) float64 {
	return min(max((p-lo)/(hi-lo), 0), 1)
}

func lerp(from, to int, p float64) int {
	return from + int(math.Round(float64(to-from)*p))
}

func frameWidth(frame string) int {
	width := 0
	for line := range strings.SplitSeq(frame, "\n") {
		width = max(width, ansi.StringWidth(line))
	}
	return width
}

// A canvas is a frame being drawn.
type canvas struct {
	width, height int
	cells         []cell
	out           []byte
}

// A cell of the canvas: the glyph on it, and how strongly it was drawn, so
// that where one piece fades out and another fades in on the same cell the
// stronger one shows.
type cell struct {
	glyph
	alpha float32
}

// reset blanks the canvas for a frame, keeping the memory of the last one.
func (c *canvas) reset(width, height int) {
	if len(c.cells) != width*height {
		c.cells = make([]cell, width*height)
	} else {
		clear(c.cells)
	}
	c.width, c.height = width, height
}

func (c *canvas) set(x, y int, g glyph, alpha float32) {
	if x < 0 || y < 0 || x >= c.width || y >= c.height {
		return
	}
	row := c.cells[y*c.width : (y+1)*c.width]
	if alpha < row[x].alpha {
		return
	}

	width := g.cells()
	if x+width > c.width {
		return
	}
	for i := x; i < x+width; i++ {
		unwide(row, i)
	}

	g.rest = false
	row[x] = cell{glyph: g, alpha: alpha}
	for i := x + 1; i < x+width; i++ {
		row[i] = cell{glyph: glyph{rest: true}, alpha: alpha}
	}
}

// unwide blanks the wide glyph that covers cell i, if there is one, so that
// drawing over half of it doesn't leave the other half behind.
func unwide(row []cell, i int) {
	head := i
	for head > 0 && row[head].rest {
		head--
	}
	width := int(row[head].width)
	if head == i && width <= 1 {
		return
	}
	for j := head; j < head+max(width, 1) && j < len(row); j++ {
		row[j].glyph = glyph{}
	}
}

// box draws the outline of a box from corner x0, y0 to corner x1, y1.
func (c *canvas) box(e edges, x0, y0, x1, y1 int, style int32, alpha float64) {
	if alpha <= 0 {
		return
	}
	e.outline(x0, y0, x1, y1, func(x, y int, text string) {
		c.set(x, y, glyph{text: text, style: style, width: 1}, float32(alpha))
	})
}

// String writes the canvas out, switching styles only where they change.
func (c *canvas) String(s *styles) string {
	c.out = c.out[:0]

	for y := range c.height {
		pen := int32(0)
		for _, cl := range c.cells[y*c.width : (y+1)*c.width] {
			switch {
			case cl.rest:
				continue
			case cl.text == "" && !s.shown(pen):
				// A blank looks the same in any style that doesn't paint
				// blanks, so there is no need to switch to its own.
				c.out = append(c.out, ' ')
				continue
			case cl.style != pen:
				c.out = s.appendSGR(c.out, cl.style)
				pen = cl.style
			}

			if cl.text == "" {
				c.out = append(c.out, ' ')
			} else {
				c.out = append(c.out, cl.text...)
			}
		}

		if pen != 0 {
			c.out = append(c.out, ansi.ResetStyle...)
		}
		if y < c.height-1 {
			c.out = append(c.out, '\n')
		}
	}

	return string(c.out)
}
