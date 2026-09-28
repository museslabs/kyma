package transitions

import (
	"image/color"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/cellbuf"
)

// slide pads every line to the same width, the way a rendered slide comes.
func slide(width int, lines ...string) string {
	padded := make([]string, len(lines))
	for i, l := range lines {
		padded[i] = l + strings.Repeat(" ", width-ansi.StringWidth(l))
	}
	return strings.Join(padded, "\n")
}

// looks is what a frame shows, cell by cell. Blanks that show nothing come out
// as plain spaces however they are styled, since they look the same.
func looks(frame string, width, height int) []string {
	buf := cellbuf.NewBuffer(width, height)
	cellbuf.SetContent(buf, frame)

	rows := make([]string, height)
	for y := range height {
		var b strings.Builder
		for x := range width {
			c := buf.Cell(x, y)
			switch {
			case c.Width == 0:
			case shows(c):
				b.WriteString(c.Style.Sequence() + c.String())
			default:
				b.WriteString(" ")
			}
		}
		rows[y] = b.String()
	}
	return rows
}

// shows reports whether a cell shows anything: a glyph, or a blank with a
// background, reverse video, an underline or a strike-through.
func shows(c *cellbuf.Cell) bool {
	if c.Rune != ' ' || len(c.Comb) > 0 {
		return c.Rune != 0
	}
	return c.Style.Bg != nil ||
		c.Style.UlStyle != cellbuf.NoUnderline ||
		c.Style.Attrs&(cellbuf.ReverseAttr|cellbuf.StrikethroughAttr) != 0
}

func text(glyphs []glyph) string {
	var b strings.Builder
	for _, g := range glyphs {
		if g.text == "" {
			b.WriteString(" ")
		}
		b.WriteString(g.text)
	}
	return b.String()
}

func moveOf(t *testing.T, s scene, want string) move {
	t.Helper()
	for _, m := range s.moving {
		if text(m.glyphs) == want {
			return m
		}
	}
	t.Fatalf("%q does not move; the moves are %v", want, moveTexts(s))
	return move{}
}

func moveTexts(s scene) []string {
	var out []string
	for _, m := range s.moving {
		out = append(out, text(m.glyphs))
	}
	return out
}

func pieceTexts(pieces []piece) []string {
	var out []string
	for _, p := range pieces {
		out = append(out, text(p.glyphs))
	}
	return out
}

var morphCases = map[string]struct {
	width, height int
	prev, next    string
}{
	"growing list": {
		width: 12, height: 8,
		prev: slide(12, "", "", "", "• one", "• two", "", "", ""),
		next: slide(12, "", "", "• one", "• two", "• three", "", "", ""),
	},
	"styled": {
		width: 30, height: 4,
		prev: slide(30,
			"\x1b[1;38;5;228;48;5;63m Title \x1b[0m",
			"",
			"some \x1b[3mitalic\x1b[0m words",
			"\x1b[4mlinked text\x1b[0m",
		),
		next: slide(30,
			"",
			"   \x1b[1;38;5;228;48;5;63m Title \x1b[0m",
			"\x1b[3mitalic\x1b[0m words, fewer",
			"",
		),
	},
	"wide characters": {
		width: 20, height: 4,
		prev: slide(20, "", " 日本語 text", "漢字 moves", ""),
		next: slide(20, "漢字", "", "    日本語 text", "   漢字 moves"),
	},
	"unrelated": {
		width: 24, height: 3,
		prev: slide(24, "the cat sat on the mat", "", ""),
		next: slide(24, "", "", "a dog ran to the park"),
	},
}

// The morph has to start exactly on the slide being left and end exactly on
// the one arriving, or the switch to the real slide would jump.
func TestMorph_StartsOnPrevAndEndsOnNext(t *testing.T) {
	for name, tc := range morphCases {
		t.Run(name, func(t *testing.T) {
			s := newScene(tc.prev, tc.next)

			for _, check := range []struct {
				progress    float64
				want, label string
			}{{0, tc.prev, "first"}, {1, tc.next, "last"}} {
				got := looks(s.frame(check.progress), tc.width, tc.height)
				want := looks(check.want, tc.width, tc.height)
				if !slices.Equal(got, want) {
					t.Errorf("the %s frame is\n%q\nwant\n%q", check.label, got, want)
				}
			}
		})
	}
}

func TestMorph_FramesKeepTheSlideSize(t *testing.T) {
	for name, tc := range morphCases {
		t.Run(name, func(t *testing.T) {
			s := newScene(tc.prev, tc.next)

			for step := range 21 {
				p := float64(step) / 20
				lines := strings.Split(s.frame(p), "\n")
				if len(lines) != tc.height {
					t.Fatalf("at %.2f the frame has %d lines, want %d", p, len(lines), tc.height)
				}
				for i, line := range lines {
					if got := ansi.StringWidth(line); got != tc.width {
						t.Errorf("at %.2f line %d is %d wide, want %d: %q", p, i, got, tc.width, line)
					}
				}
			}
		})
	}
}

func TestMorph_GrowingListSlidesUp(t *testing.T) {
	tc := morphCases["growing list"]
	s := newScene(tc.prev, tc.next)

	for _, want := range []struct {
		text     string
		from, to int
	}{{"• one", 3, 2}, {"• two", 4, 3}} {
		m := moveOf(t, s, want.text)
		if m.x != 0 || m.toX != 0 || m.y != want.from || m.toY != want.to {
			t.Errorf("%q moves from %d,%d to %d,%d, want from 0,%d to 0,%d",
				want.text, m.x, m.y, m.toX, m.toY, want.from, want.to)
		}
	}

	if got := pieceTexts(s.entering); !slices.Equal(got, []string{"•", "three"}) {
		t.Errorf("entering %q, want the new item", got)
	}
	if len(s.leaving) > 0 {
		t.Errorf("leaving %q, want nothing", pieceTexts(s.leaving))
	}
}

func TestMorph_BorderStaysPut(t *testing.T) {
	const width, height = 12, 8
	prev := slide(width,
		"╭──────────╮",
		"│          │",
		"│          │",
		"│  Title   │",
		"│  • one   │",
		"│          │",
		"│          │",
		"╰──────────╯",
	)
	next := slide(width,
		"╭──────────╮",
		"│          │",
		"│  Title   │",
		"│  • one   │",
		"│  • two   │",
		"│          │",
		"│          │",
		"╰──────────╯",
	)
	s := newScene(prev, next)

	if m := moveOf(t, s, "Title"); m.toY != 2 {
		t.Errorf("the title moves to row %d, want 2", m.toY)
	}

	for step := range 21 {
		p := float64(step) / 20
		rows := strings.Split(ansi.Strip(s.frame(p)), "\n")
		if rows[0] != "╭──────────╮" || rows[height-1] != "╰──────────╯" {
			t.Fatalf("at %.2f the border broke:\n%s", p, strings.Join(rows, "\n"))
		}
		for _, row := range rows[1 : height-1] {
			if !strings.HasPrefix(row, "│") || !strings.HasSuffix(row, "│") {
				t.Fatalf("at %.2f the border broke:\n%s", p, strings.Join(rows, "\n"))
			}
		}
	}
}

// Lines that repeat, like the closing braces of code, have to move along with
// the code around them rather than trade places with one another.
func TestMorph_RepeatedLinesMoveTogether(t *testing.T) {
	prev := slide(3, "a {", "}", "b {", "}", "c {", "}")
	next := slide(3, "x", "y", "a {", "}", "b {", "}", "c {", "}")
	s := newScene(prev, next)

	if len(s.moving) != 6 {
		t.Fatalf("moving %q, want all six lines", moveTexts(s))
	}
	for _, m := range s.moving {
		if m.toX != m.x || m.toY != m.y+2 {
			t.Errorf("%q moves from %d,%d to %d,%d, want two rows down",
				text(m.glyphs), m.x, m.y, m.toX, m.toY)
		}
	}
}

func TestMorph_EditedLineKeepsItsWords(t *testing.T) {
	prev := slide(24, "", "x := compute(a)", "")
	next := slide(24, "", "x, err := compute(a)", "")
	s := newScene(prev, next)

	for _, want := range []struct {
		text     string
		from, to int
	}{{":=", 2, 7}, {"compute(a)", 5, 10}} {
		m := moveOf(t, s, want.text)
		if m.x != want.from || m.toX != want.to || m.y != 1 || m.toY != 1 {
			t.Errorf("%q moves from %d,%d to %d,%d, want from %d,1 to %d,1",
				want.text, m.x, m.y, m.toX, m.toY, want.from, want.to)
		}
	}
	if got := pieceTexts(s.leaving); !slices.Equal(got, []string{"x"}) {
		t.Errorf("leaving %q, want the old word", got)
	}
	if got := pieceTexts(s.entering); !slices.Equal(got, []string{"x,", "err"}) {
		t.Errorf("entering %q, want the new words", got)
	}
}

// Code is split into its highlighted tokens, so an argument added to a call
// keeps the rest of the call.
func TestMorph_EditedCodeKeepsItsTokens(t *testing.T) {
	const fn, str, name, punct = "\x1b[32m", "\x1b[33m", "\x1b[36m", "\x1b[37m"
	prev := slide(40, fn+"fmt.Println"+punct+"("+str+`"hi"`+punct+")\x1b[0m")
	next := slide(40, fn+"fmt.Println"+punct+"("+str+`"hi"`+punct+", "+name+"name"+punct+")\x1b[0m")
	s := newScene(prev, next)

	if m := moveOf(t, s, ")"); m.x != 16 || m.toX != 22 {
		t.Errorf("the closing paren moves from %d to %d, want from 16 to 22", m.x, m.toX)
	}
	if got := pieceTexts(s.entering); !slices.Equal(got, []string{",", "name"}) {
		t.Errorf("entering %q, want the new argument", got)
	}
	for _, same := range []string{"fmt.Println", "(", `"hi"`} {
		if m := moveOf(t, s, same); m.x != m.toX {
			t.Errorf("%q moves from %d to %d, want it to stay", same, m.x, m.toX)
		}
	}
}

// Lines with little in common fade instead of sending their small words
// flying across the slide.
func TestMorph_UnrelatedLinesDissolve(t *testing.T) {
	tc := morphCases["unrelated"]
	s := newScene(tc.prev, tc.next)

	if len(s.moving) > 0 {
		t.Errorf("moving %q, want nothing", moveTexts(s))
	}
}

// frameCount is how many frames a morph takes at the rate it animates at.
var frameCount = int(morphDuration * RefreshRate / time.Second)

// progressOf is the progress at the nth of the frames.
func progressOf(n int) float64 {
	return float64(n) / float64(frameCount)
}

func TestMorph_MovesGlide(t *testing.T) {
	prev := slide(10, "", "", "", "", "", "Title")
	next := slide(10, "Title", "", "", "", "", "")
	s := newScene(prev, next)

	var rows []int
	for n := range frameCount + 1 {
		for y, row := range strings.Split(ansi.Strip(s.frame(progressOf(n))), "\n") {
			if strings.Contains(row, "Title") {
				rows = append(rows, y)
			}
		}
	}

	if len(rows) != frameCount+1 {
		t.Fatalf("the title shows up %d times over %d frames, want once a frame", len(rows), frameCount+1)
	}
	if got := slices.Compact(slices.Clone(rows)); !slices.Equal(got, []int{5, 4, 3, 2, 1, 0}) {
		t.Errorf("the title goes through rows %v, want every row from 5 up to 0", got)
	}
}

// at is a cell of a frame and where it is.
type at struct {
	x, y int
	cellbuf.Cell
}

// find returns the cells of a frame that show r.
func find(frame string, width, height int, r rune) []at {
	buf := cellbuf.NewBuffer(width, height)
	cellbuf.SetContent(buf, frame)

	var out []at
	for y := range height {
		for x := range width {
			if c := buf.Cell(x, y); c.Rune == r {
				out = append(out, at{x, y, *c})
			}
		}
	}
	return out
}

func lightness(c ansi.Color) float64 {
	v, ok := makeColor(c)
	if !ok {
		return -1
	}
	l, _, _ := v.OkLab()
	return l
}

// What only one slide has fades into the background and out of it, rather
// than popping in or out.
func TestMorph_Fades(t *testing.T) {
	const width = 10
	prev := slide(width, "\x1b[38;2;255;255;255m"+strings.Repeat("a", width)+"\x1b[0m", "")
	next := slide(width, "", "\x1b[38;2;255;255;255m"+strings.Repeat("b", width)+"\x1b[0m")
	s := newScene(prev, next)

	lastA, lastB := 2.0, -1.0
	var between bool
	for n := range frameCount + 1 {
		frame := s.frame(progressOf(n))

		a, b := find(frame, width, 2, 'a'), find(frame, width, 2, 'b')
		if len(a) != 0 && len(a) != width || len(b) != 0 && len(b) != width {
			t.Fatalf("frame %d has %d a's and %d b's, want every cell of a word or none", n, len(a), len(b))
		}

		la, lb := -1.0, -1.0
		if len(a) > 0 {
			la = lightness(a[0].Style.Fg)
		}
		if len(b) > 0 {
			lb = lightness(b[0].Style.Fg)
		}
		if la > lastA || lb < lastB {
			t.Fatalf("frame %d: a is at %.2f after %.2f and b at %.2f after %.2f, want a to dim and b to brighten",
				n, la, lastA, lb, lastB)
		}
		lastA, lastB = la, lb

		between = between || (la > 0 && la < 0.99) || (lb > 0 && lb < 0.99)
	}

	if lastA != -1 || lastB < 0.99 {
		t.Errorf("the morph ends with a at %.2f and b at %.2f, want a gone and b at full", lastA, lastB)
	}
	if !between {
		t.Error("the words pop in and out, want them to fade")
	}
}

// Where one piece fades out and another fades into the same cell, the cell
// goes straight from one to the other.
func TestMorph_CrossFadesInPlace(t *testing.T) {
	s := newScene(slide(3, " 1 "), slide(3, " 2 "))

	var seen []string
	for n := range frameCount + 1 {
		seen = append(seen, strings.TrimSpace(ansi.Strip(s.frame(progressOf(n)))))
	}

	if got := slices.Compact(seen); !slices.Equal(got, []string{"1", "2"}) {
		t.Errorf("the cell shows %q, want 1 and then 2", got)
	}
}

// box draws a box of a color around text, a line for each row.
func box(color string, width int, text string) []string {
	inner := width - 2
	return []string{
		color + "╭" + strings.Repeat("─", inner) + "╮\x1b[0m",
		color + "│\x1b[0m " + text + strings.Repeat(" ", inner-1-len(text)) + color + "│\x1b[0m",
		color + "╰" + strings.Repeat("─", inner) + "╯\x1b[0m",
	}
}

// beside puts boxes next to each other, gap cells apart.
func beside(gap int, boxes ...[]string) []string {
	rows := make([]string, 3)
	for i, b := range boxes {
		for y := range rows {
			if i > 0 {
				rows[y] += strings.Repeat(" ", gap)
			}
			rows[y] += b[y]
		}
	}
	return rows
}

// outline finds the box drawn on a frame, from its corners.
func outline(t *testing.T, frame string, width, height int) (x0, y0, x1, y1 int) {
	t.Helper()

	var corners [4]at
	for i, r := range []rune("╭╮╰╯") {
		found := find(frame, width, height, r)
		if len(found) != 1 {
			t.Fatalf("the frame has %d of %q, want one:\n%s", len(found), r, ansi.Strip(frame))
		}
		corners[i] = found[0]
	}

	tl, tr, bl, br := corners[0], corners[1], corners[2], corners[3]
	if tl.y != tr.y || bl.y != br.y || tl.x != bl.x || tr.x != br.x {
		t.Fatalf("the corners don't make a box:\n%s", ansi.Strip(frame))
	}
	return tl.x, tl.y, br.x, br.y
}

func TestMorph_BoxesMoveWhole(t *testing.T) {
	const width, height = 24, 6
	b := box("\x1b[35m", 8, "hi")
	prev := slide(width, append(b, "", "", "")...)
	next := slide(width, "", "", "", "                "+b[0], "                "+b[1], "                "+b[2])
	s := newScene(prev, next)

	lastX, lastY := 0, 0
	for n := range frameCount + 1 {
		frame := s.frame(progressOf(n))

		x0, y0, x1, y1 := outline(t, frame, width, height)
		if x1-x0 != 7 || y1-y0 != 2 {
			t.Fatalf("frame %d: the box is %dx%d, want 8x3", n, x1-x0+1, y1-y0+1)
		}
		if x0 < lastX || y0 < lastY {
			t.Fatalf("frame %d: the box went back from %d,%d to %d,%d", n, lastX, lastY, x0, y0)
		}
		lastX, lastY = x0, y0

		if hi := strings.Split(ansi.Strip(frame), "\n")[y0+1]; !strings.HasPrefix(hi[x0+4:], "hi") {
			t.Fatalf("frame %d: the text came out of the box:\n%s", n, ansi.Strip(frame))
		}
	}
	if lastX != 16 || lastY != 3 {
		t.Errorf("the box ends at %d,%d, want 16,3", lastX, lastY)
	}
}

// Boxes are told apart by what they hold, so two boxes of the same size that
// swap places and colors each carry their own text across.
func TestMorph_BoxesKeepTheirContent(t *testing.T) {
	const width, height = 20, 3
	const red, blue, green, yellow = "\x1b[31m", "\x1b[34m", "\x1b[32m", "\x1b[33m"
	prev := slide(width, beside(2, box(red, 8, "A"), box(blue, 8, "B"))...)
	next := slide(width, beside(2, box(green, 8, "B"), box(yellow, 8, "A"))...)
	s := newScene(prev, next)

	if len(s.boxes) != 2 {
		t.Fatalf("the scene has %d boxes, want 2", len(s.boxes))
	}
	for _, b := range s.boxes {
		if b.from == nil || b.to == nil {
			t.Fatalf("a box has nothing to turn into")
		}
		if b.from.x0 == b.to.x0 {
			t.Errorf("the box at %d stays put, want it to swap places with the other", b.from.x0)
		}
		if b.from.style == b.to.style {
			t.Errorf("the box at %d keeps its color, want it to change", b.from.x0)
		}
	}
	for _, text := range []string{"A", "B"} {
		if m := moveOf(t, s, text); m.x == m.toX {
			t.Errorf("%s stays put, want it to go with its box", text)
		}
	}
}

func TestMorph_BoxesResize(t *testing.T) {
	const width, height = 16, 5
	small := box("\x1b[36m", 6, "hi")
	big := []string{
		"\x1b[36m╭──────────────╮\x1b[0m",
		"\x1b[36m│\x1b[0m hi           \x1b[36m│\x1b[0m",
		"\x1b[36m│\x1b[0m              \x1b[36m│\x1b[0m",
		"\x1b[36m│\x1b[0m              \x1b[36m│\x1b[0m",
		"\x1b[36m╰──────────────╯\x1b[0m",
	}
	s := newScene(slide(width, append(small, "", "")...), slide(width, big...))

	var sizes [][2]int
	for n := range frameCount + 1 {
		x0, y0, x1, y1 := outline(t, s.frame(progressOf(n)), width, height)
		if x0 != 0 || y0 != 0 {
			t.Fatalf("frame %d: the box moved to %d,%d, want it to grow from its corner", n, x0, y0)
		}
		sizes = append(sizes, [2]int{x1 + 1, y1 + 1})
	}

	sizes = slices.Compact(sizes)
	if first, last := sizes[0], sizes[len(sizes)-1]; first != [2]int{6, 3} || last != [2]int{16, 5} {
		t.Errorf("the box goes from %v to %v, want from 6x3 to 16x5", first, last)
	}
	if len(sizes) < 5 {
		t.Errorf("the box grows through %v, want it to grow a bit at a time", sizes)
	}
}

// Text that only changed color moves like any other, and takes on its new
// color on the way.
func TestMorph_RecoloredTextMoves(t *testing.T) {
	const width, height = 10, 4
	const red, blue = "\x1b[38;2;255;0;0m", "\x1b[38;2;0;0;255m"
	s := newScene(
		slide(width, "", "", "", red+"Title\x1b[0m"),
		slide(width, blue+"Title\x1b[0m", "", "", ""),
	)

	m := moveOf(t, s, "Title")
	if !m.restyled || m.y != 3 || m.toY != 0 {
		t.Fatalf("Title moves from row %d to %d, restyled %v; want from 3 to 0, restyled", m.y, m.toY, m.restyled)
	}

	mid := find(s.frame(0.5), width, height, 'T')
	if len(mid) != 1 {
		t.Fatalf("the title shows up %d times halfway, want once", len(mid))
	}
	r, _, b, _ := mid[0].Style.Fg.RGBA()
	if r == 0 || b == 0 {
		t.Errorf("halfway the title is %v, want a blend of red and blue", mid[0].Style.Fg)
	}
}

// Fades go to the terminal's own background, whatever it is.
func TestMorph_FadesToTheTerminalBackground(t *testing.T) {
	fg, bg := terminalFg, terminalBg
	t.Cleanup(func() { terminalFg, terminalBg = fg, bg })
	SetTerminalColors(nil, color.White)

	s := newScene(slide(4, "\x1b[38;2;0;0;0mgone\x1b[0m"), slide(4, ""))
	cells := find(s.frame(0.3), 4, 1, 'g')
	if len(cells) != 1 {
		t.Fatalf("the word is gone already")
	}
	if l := lightness(cells[0].Style.Fg); l <= 0.05 {
		t.Errorf("black text fading on white is at lightness %.2f, want it lighter", l)
	}
}

// The morph runs on the clock, starting from its first frame.
func TestMorph_RunsOnTheClock(t *testing.T) {
	clock := time.Unix(0, 0)
	now = func() time.Time { return clock }
	t.Cleanup(func() { now = time.Now })

	var tr Transition = newMorph(Fps).Start(10, 2, Forwards)
	clock = clock.Add(time.Second)

	tr, _ = tr.Update()
	if got := tr.(morph).progress; got != 0 {
		t.Errorf("the first frame is at %.2f, want 0: the clock starts there", got)
	}

	clock = clock.Add(morphDuration / 2)
	tr, _ = tr.Update()
	if got := tr.(morph).progress; math.Abs(got-0.5) > 1e-9 || !tr.Animating() {
		t.Errorf("halfway through the progress is %.2f, want 0.5", got)
	}

	clock = clock.Add(morphDuration)
	tr, cmd := tr.Update()
	if tr.Animating() || cmd != nil || tr.(morph).progress != 1 {
		t.Errorf("past the end the morph is still going")
	}
}

// The scene is kept between frames, but only for the slides it was made for.
func TestMorph_ViewFollowsTheSlides(t *testing.T) {
	m := newMorph(Fps).Start(10, 2, Forwards).(morph)
	m.progress = 1

	first := slide(10, "one", "")
	second := slide(10, "", "two")
	third := slide(10, "three", "")

	if got, want := looks(m.View(first, second), 10, 2), looks(second, 10, 2); !slices.Equal(got, want) {
		t.Errorf("View() = %q, want %q", got, want)
	}
	if got, want := looks(m.View(second, third), 10, 2), looks(third, 10, 2); !slices.Equal(got, want) {
		t.Errorf("View() after the slides changed = %q, want %q", got, want)
	}
}

func TestGet_Morph(t *testing.T) {
	if got := Get("morph", Fps).Name(); got != "morph" {
		t.Errorf("Get(\"morph\").Name() = %q", got)
	}
}
