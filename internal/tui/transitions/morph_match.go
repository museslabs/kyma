package transitions

import (
	"cmp"
	"slices"
	"strings"
	"unicode/utf8"
)

// visible reports whether a glyph shows anything: a grapheme other than a
// blank, or a blank in a style that paints it, with a background, reverse
// video, an underline or a strike-through.
func (s *scene) visible(g glyph) bool {
	if g.rest {
		return false
	}
	if g.text != "" && g.text != " " {
		return true
	}
	return s.styles.shows[g.style]
}

// phrases breaks a slide into runs of visible glyphs, top to bottom. A single
// blank doesn't end a run, so the words of a line stay together, while the
// wider gaps between columns or before an indented line split it.
func (s *scene) phrases(grid []glyph) []piece {
	var out []piece

	for y := range s.height {
		var run []glyph
		start, gap := 0, 0

		for x, g := range grid[y*s.width : (y+1)*s.width] {
			switch {
			case g.rest:
				// The right half of a wide glyph, which the run already has.
			case !s.visible(g):
				gap++
			case len(run) > 0 && gap <= 1:
				if gap == 1 {
					run = append(run, glyph{})
				}
				run = append(run, g)
				gap = 0
			default:
				if len(run) > 0 {
					out = append(out, newPiece(start, y, run))
				}
				run, start, gap = []glyph{g}, x, 0
			}
		}

		if len(run) > 0 {
			out = append(out, newPiece(start, y, run))
		}
	}

	return out
}

// splitWords breaks the pieces that have no counterpart into words, noting
// which piece each word came from.
func (s *scene) splitWords(pieces []piece, counterpart []int) (words []piece, from []int) {
	for i, p := range pieces {
		if counterpart[i] >= 0 {
			continue
		}
		for _, w := range s.words(p) {
			words = append(words, w)
			from = append(from, i)
		}
	}
	return words, from
}

// words breaks a piece at its blanks and wherever its style changes, which in
// highlighted code is between tokens.
func (s *scene) words(p piece) []piece {
	var out []piece
	start, startX := -1, 0

	x := p.x
	for i, g := range p.glyphs {
		if start >= 0 && (!s.visible(g) || g.style != p.glyphs[start].style) {
			out = append(out, newPiece(startX, p.y, p.glyphs[start:i]))
			start = -1
		}
		if start < 0 && s.visible(g) {
			start, startX = i, x
		}
		x += g.cells()
	}

	if start >= 0 {
		out = append(out, newPiece(startX, p.y, p.glyphs[start:]))
	}
	return out
}

func newPiece(x, y int, glyphs []glyph) piece {
	// Each glyph goes into the key as its style's number and its text, and
	// into the shape as its text, each ended by a zero byte that no text has,
	// so two pieces only share a key or a shape glyph for glyph.
	var key, shape []byte
	for _, g := range glyphs {
		key = append(key, byte(g.style), byte(g.style>>8), byte(g.style>>16), byte(g.style>>24))
		key = append(key, g.text...)
		key = append(key, 0)
		if g.text == "" {
			shape = append(shape, ' ')
		}
		shape = append(shape, g.text...)
		shape = append(shape, 0)
	}
	return piece{x: x, y: y, glyphs: glyphs, key: string(key), shape: string(shape)}
}

func (p piece) width() int {
	width := 0
	for _, g := range p.glyphs {
		width += g.cells()
	}
	return width
}

// A frame is a box drawn with line characters, like the slide's border or a
// bordered column.
type frame struct {
	// The corners, where x1, y1 is the bottom right one.
	x0, y0, x1, y1 int
	edges          edges
	style          int32
}

// edges are the characters a frame is drawn with.
type edges struct {
	topLeft, top, topRight          string
	left, right                     string
	bottomLeft, bottom, bottomRight string
}

// outline calls draw for every cell of a box's outline with the character that
// goes there.
func (e edges) outline(x0, y0, x1, y1 int, draw func(x, y int, text string)) {
	for x := x0 + 1; x < x1; x++ {
		draw(x, y0, e.top)
		draw(x, y1, e.bottom)
	}
	for y := y0 + 1; y < y1; y++ {
		draw(x0, y, e.left)
		draw(x1, y, e.right)
	}
	draw(x0, y0, e.topLeft)
	draw(x1, y0, e.topRight)
	draw(x0, y1, e.bottomLeft)
	draw(x1, y1, e.bottomRight)
}

// The characters of the borders a slide can be drawn with, rounded, normal,
// double and thick.
const (
	topLefts     = "╭┌╔┏"
	topRights    = "╮┐╗┓"
	bottomLefts  = "╰└╚┗"
	bottomRights = "╯┘╝┛"
	horizontals  = "─━═"
	verticals    = "│┃║"
)

// frames finds the boxes drawn on a slide and lifts them off it, leaving the
// rest to be broken into pieces.
func (s *scene) frames(grid []glyph) []frame {
	var out []frame
	for i := range grid {
		// Line characters all start with the same byte in UTF-8, which
		// rules out most cells before looking any closer.
		if text := grid[i].text; len(text) != 3 || text[0] != 0xE2 {
			continue
		}
		if f, ok := s.frameAt(grid, i%s.width, i/s.width); ok {
			out = append(out, f)
		}
	}

	for _, f := range out {
		f.edges.outline(f.x0, f.y0, f.x1, f.y1, func(x, y int, _ string) {
			grid[y*s.width+x] = glyph{}
		})
	}
	return out
}

// frameAt reads the box whose top left corner is at x0, y0, if there is one:
// a closed outline of line characters in a single style.
func (s *scene) frameAt(grid []glyph, x0, y0 int) (frame, bool) {
	corner := grid[y0*s.width+x0]
	if !oneOf(topLefts, corner.text) {
		return frame{}, false
	}
	f := frame{x0: x0, y0: y0, style: corner.style}
	f.edges.topLeft = corner.text

	// edge reads a cell of the outline, which has to be drawn in the frame's
	// style with one of the characters in set. The first character read for
	// an edge is the one the rest of it has to be.
	edge := func(x, y int, set string, text *string) bool {
		if x >= s.width || y >= s.height {
			return false
		}
		g := grid[y*s.width+x]
		if g.style != f.style || !oneOf(set, g.text) {
			return false
		}
		if *text == "" {
			*text = g.text
		}
		return g.text == *text
	}

	// Along the top to the other corner, and down the left side to the
	// bottom, which gives the size.
	f.x1 = x0 + 1
	for edge(f.x1, y0, horizontals, &f.edges.top) {
		f.x1++
	}
	f.y1 = y0 + 1
	for edge(x0, f.y1, verticals, &f.edges.left) {
		f.y1++
	}
	if f.x1 < x0+2 || f.y1 < y0+2 ||
		!edge(f.x1, y0, topRights, &f.edges.topRight) ||
		!edge(x0, f.y1, bottomLefts, &f.edges.bottomLeft) ||
		!edge(f.x1, f.y1, bottomRights, &f.edges.bottomRight) {
		return frame{}, false
	}

	// The right side and the bottom have to close the box.
	for y := y0 + 1; y < f.y1; y++ {
		if !edge(f.x1, y, verticals, &f.edges.right) {
			return frame{}, false
		}
	}
	for x := x0 + 1; x < f.x1; x++ {
		if !edge(x, f.y1, horizontals, &f.edges.bottom) {
			return frame{}, false
		}
	}

	return f, true
}

// oneOf reports whether text is a single one of the characters in set.
func oneOf(set, text string) bool {
	r, n := utf8.DecodeRuneInString(text)
	return n > 0 && n == len(text) && strings.ContainsRune(set, r)
}

// contains reports whether a run of cells, width wide from x, y, is inside the
// frame's outline.
func (f frame) contains(x, y, width int) bool {
	return f.x0 < x && x+width-1 < f.x1 && f.y0 < y && y < f.y1
}

func (f frame) area() int {
	return (f.x1 - f.x0 + 1) * (f.y1 - f.y0 + 1)
}

// A boxMove is a frame travelling from where the previous slide has it to
// where the next one does. When only one slide has it, the other end is nil
// and the frame fades instead.
type boxMove struct {
	from, to *frame
}

func (b boxMove) area() int {
	if b.from != nil {
		return b.from.area()
	}
	return b.to.area()
}

// matchFrames pairs up the boxes of the two slides. A box is told apart by
// what is in it: the one that holds the same text on both slides is the same
// box, wherever it went and whatever color it turned. Boxes with nothing in
// common pair up by size and then by how close they are, and morph from one
// into the other.
func matchFrames(a, b []frame, moves []move) []boxMove {
	shared := make(map[[2]int]int)
	for _, m := range moves {
		width := m.width()
		i, j := innermost(a, m.x, m.y, width), innermost(b, m.toX, m.toY, width)
		if i >= 0 && j >= 0 {
			shared[[2]int{i, j}]++
		}
	}

	type candidate struct {
		i, j, shared int
		sameSize     bool
		dist         int
	}

	var candidates []candidate
	for i, f := range a {
		for j, g := range b {
			candidates = append(candidates, candidate{
				i:        i,
				j:        j,
				shared:   shared[[2]int{i, j}],
				sameSize: f.x1-f.x0 == g.x1-g.x0 && f.y1-f.y0 == g.y1-g.y0,
				dist:     distance(f.x0, f.y0, g.x0, g.y0) + distance(f.x1, f.y1, g.x1, g.y1),
			})
		}
	}

	slices.SortStableFunc(candidates, func(p, q candidate) int {
		if p.sameSize != q.sameSize && p.shared == q.shared {
			if p.sameSize {
				return -1
			}
			return 1
		}
		return cmp.Or(cmp.Compare(q.shared, p.shared), cmp.Compare(p.dist, q.dist))
	})

	var out []boxMove
	pairedA, pairedB := make([]bool, len(a)), make([]bool, len(b))
	for _, c := range candidates {
		if pairedA[c.i] || pairedB[c.j] {
			continue
		}
		pairedA[c.i], pairedB[c.j] = true, true
		out = append(out, boxMove{&a[c.i], &b[c.j]})
	}
	for i := range a {
		if !pairedA[i] {
			out = append(out, boxMove{from: &a[i]})
		}
	}
	for j := range b {
		if !pairedB[j] {
			out = append(out, boxMove{to: &b[j]})
		}
	}

	// Outer boxes are drawn first, so the ones inside them go on top.
	slices.SortStableFunc(out, func(p, q boxMove) int {
		return cmp.Compare(q.area(), p.area())
	})
	return out
}

// innermost returns the smallest of the frames around a run of cells, or -1.
func innermost(frames []frame, x, y, width int) int {
	found := -1
	for i, f := range frames {
		if f.contains(x, y, width) && (found < 0 || f.area() < frames[found].area()) {
			found = i
		}
	}
	return found
}

// A matching pairs pieces of the previous slide, a, with pieces of the next
// one, b, that look the same. ai[i] is the piece of b that a[i] turns into and
// bi[j] the piece of a that b[j] comes from, or -1 for none.
type matching struct {
	a, b   []piece
	ai, bi []int
	// byShape makes pieces pair up by their text alone, so what only changed
	// color is still found.
	byShape bool
}

func newMatching(a, b []piece) *matching {
	m := &matching{a: a, b: b, ai: make([]int, len(a)), bi: make([]int, len(b))}
	for i := range m.ai {
		m.ai[i] = -1
	}
	for j := range m.bi {
		m.bi[j] = -1
	}
	return m
}

func (m *matching) key(p piece) string {
	if m.byShape {
		return p.shape
	}
	return p.key
}

func (m *matching) pair(i, j int) {
	m.ai[i], m.bi[j] = j, i
}

// unmatched returns the pieces of a and of b that have no counterpart yet.
func (m *matching) unmatched() (as, bs []int) {
	for i, j := range m.ai {
		if j < 0 {
			as = append(as, i)
		}
	}
	for j, i := range m.bi {
		if i < 0 {
			bs = append(bs, j)
		}
	}
	return as, bs
}

// moves returns the pairs as moves from a to b.
func (m *matching) moves() []move {
	var out []move
	for i, j := range m.ai {
		if j < 0 {
			continue
		}
		out = append(out, move{
			piece:    m.a[i],
			toX:      m.b[j].x,
			toY:      m.b[j].y,
			to:       m.b[j].glyphs,
			restyled: m.a[i].key != m.b[j].key,
		})
	}
	return out
}

// inPlace pairs the pieces that stayed exactly where they were.
func (m *matching) inPlace() {
	type spot struct {
		x, y int
		key  string
	}

	at := make(map[spot]int, len(m.b))
	for j, p := range m.b {
		if m.bi[j] < 0 {
			at[spot{p.x, p.y, m.key(p)}] = j
		}
	}

	for i, p := range m.a {
		j, ok := at[spot{p.x, p.y, m.key(p)}]
		if ok && m.ai[i] < 0 && m.bi[j] < 0 {
			m.pair(i, j)
		}
	}
}

// inOrder pairs the pieces among as and bs that kept their order, a longest
// common subsequence of the two read top to bottom. Lines pushed down by one
// inserted above them still line up with themselves this way, even where the
// same line shows up more than once, like the closing braces of code.
func (m *matching) inOrder(as, bs []int) {
	keysA := make([]string, len(as))
	for k, i := range as {
		keysA[k] = m.key(m.a[i])
	}
	keysB := make([]string, len(bs))
	for k, j := range bs {
		keysB[k] = m.key(m.b[j])
	}

	for _, p := range lcs(keysA, keysB) {
		m.pair(as[p[0]], bs[p[1]])
	}
}

// nearest pairs whatever is left with its closest lookalike, which catches the
// pieces that moved past others, like a heading swapping places with a list.
func (m *matching) nearest() {
	type candidate struct{ i, j, dist int }

	lookalikes := make(map[string][]int)
	for j, p := range m.b {
		if m.bi[j] < 0 {
			lookalikes[m.key(p)] = append(lookalikes[m.key(p)], j)
		}
	}

	var candidates []candidate
	for i, p := range m.a {
		if m.ai[i] >= 0 {
			continue
		}
		for _, j := range lookalikes[m.key(p)] {
			q := m.b[j]
			candidates = append(candidates, candidate{i, j, distance(p.x, p.y, q.x, q.y)})
		}
	}

	slices.SortStableFunc(candidates, func(x, y candidate) int {
		return cmp.Compare(x.dist, y.dist)
	})
	for _, c := range candidates {
		if m.ai[c.i] < 0 && m.bi[c.j] < 0 {
			m.pair(c.i, c.j)
		}
	}
}

// similarLines is how much two lines need in common, as the share of their
// words both have, before their words are matched with each other.
const similarLines = 0.5

// inSimilarLines pairs up the lines that changed but are still mostly the
// same line, like an edited line of code, and matches their words in order.
// Lines with less in common are left to fade, or the small words every line
// has would fly across unrelated slides. fromA and fromB tell which line each
// word of a and of b came from.
func (m *matching) inSimilarLines(fromA, fromB []int) {
	linesA, linesB := lines(m.a, m.ai, fromA), lines(m.b, m.bi, fromB)

	type candidate struct {
		x, y  int
		score float64
		dist  int
	}

	var candidates []candidate
	for x, l := range linesA {
		for y, r := range linesB {
			total := float64(len(l.keys) + len(r.keys))
			// A cheap bound first: lines of very different lengths can't
			// share enough whatever they say.
			if 2*float64(min(len(l.keys), len(r.keys))) < similarLines*total {
				continue
			}

			score := 2 * float64(shared(l.keys, r.keys)) / total
			if score >= similarLines {
				p, q := m.a[l.words[0]], m.b[r.words[0]]
				candidates = append(candidates, candidate{x, y, score, distance(p.x, p.y, q.x, q.y)})
			}
		}
	}

	slices.SortStableFunc(candidates, func(p, q candidate) int {
		return cmp.Or(cmp.Compare(q.score, p.score), cmp.Compare(p.dist, q.dist))
	})

	pairedA, pairedB := make([]bool, len(linesA)), make([]bool, len(linesB))
	for _, c := range candidates {
		if pairedA[c.x] || pairedB[c.y] {
			continue
		}
		pairedA[c.x], pairedB[c.y] = true, true
		m.inOrder(linesA[c.x].words, linesB[c.y].words)
	}
}

// A line is the words of a piece that are still unmatched, with their keys
// sorted to count what two lines share.
type line struct {
	words []int
	keys  []string
}

func lines(words []piece, counterpart, from []int) []line {
	var out []line

	last := -1
	for k, w := range words {
		if counterpart[k] >= 0 {
			continue
		}
		if len(out) == 0 || from[k] != last {
			out = append(out, line{})
			last = from[k]
		}
		l := &out[len(out)-1]
		l.words = append(l.words, k)
		l.keys = append(l.keys, w.key)
	}

	for i := range out {
		slices.Sort(out[i].keys)
	}
	return out
}

// shared counts the keys two sorted lists have in common.
func shared(a, b []string) int {
	n := 0
	for i, j := 0, 0; i < len(a) && j < len(b); {
		switch {
		case a[i] == b[j]:
			n++
			i++
			j++
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}
	return n
}

// maxLCS bounds the table lcs fills in. Slides come nowhere near it, and past
// it the pieces are left to be paired by distance alone.
const maxLCS = 1 << 20

// lcs returns the index pairs of a longest common subsequence of a and b.
func lcs(a, b []string) [][2]int {
	n, m := len(a), len(b)
	if n == 0 || m == 0 || n*m > maxLCS {
		return nil
	}

	// Comparing numbers is a lot cheaper than comparing keys in the loop.
	ids := make(map[string]int32)
	id := func(key string) int32 {
		v, ok := ids[key]
		if !ok {
			v = int32(len(ids))
			ids[key] = v
		}
		return v
	}
	x, y := make([]int32, n), make([]int32, m)
	for i, key := range a {
		x[i] = id(key)
	}
	for j, key := range b {
		y[j] = id(key)
	}

	// length[i*w+j] is the length of the longest common subsequence of
	// x[i:] and y[j:].
	w := m + 1
	length := make([]int32, (n+1)*w)
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if x[i] == y[j] {
				length[i*w+j] = length[(i+1)*w+j+1] + 1
			} else {
				length[i*w+j] = max(length[(i+1)*w+j], length[i*w+j+1])
			}
		}
	}

	var pairs [][2]int
	for i, j := 0, 0; i < n && j < m; {
		switch {
		case x[i] == y[j]:
			pairs = append(pairs, [2]int{i, j})
			i++
			j++
		case length[(i+1)*w+j] >= length[i*w+j+1]:
			i++
		default:
			j++
		}
	}
	return pairs
}

// settle trades destinations between moves of lookalike pieces for as long as
// that shortens their travel, counting one long trip as worse than two short
// ones. Otherwise a piece that repeats, like a closing brace, can stay put
// while its twin flies across the slide to fill in for the one that moved.
func settle(moves []move) {
	lookalikes := make(map[string][]*move)
	for i := range moves {
		m := &moves[i]
		lookalikes[m.key] = append(lookalikes[m.key], m)
	}

	for _, group := range lookalikes {
		for traded := true; traded; {
			traded = false
			for x, p := range group {
				for _, q := range group[x+1:] {
					now := distance(p.x, p.y, p.toX, p.toY) + distance(q.x, q.y, q.toX, q.toY)
					swapped := distance(p.x, p.y, q.toX, q.toY) + distance(q.x, q.y, p.toX, p.toY)
					if swapped < now {
						p.toX, q.toX = q.toX, p.toX
						p.toY, q.toY = q.toY, p.toY
						p.to, q.to = q.to, p.to
						p.restyled, q.restyled = q.restyled, p.restyled
						traded = true
					}
				}
			}
		}
	}
}

// distance is the squared distance between two cells, counting a row as two
// columns since a terminal cell is about twice as tall as it is wide.
func distance(x0, y0, x1, y1 int) int {
	dx, dy := x0-x1, 2*(y0-y1)
	return dx*dx + dy*dy
}
