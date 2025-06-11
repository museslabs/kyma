package transitions

import (
	"math"
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
	charmansi "github.com/charmbracelet/x/ansi"
	"github.com/muesli/reflow/truncate"

	"github.com/museslabs/kyma/internal/skip"
)

type tileGrid struct {
	tileSize   int
	gridWidth  int
	gridHeight int
	tileOrder  []int
}

func newTileGrid(width, height, tileSize int) tileGrid {
	gridWidth := (width + tileSize - 1) / tileSize
	gridHeight := (height + tileSize - 1) / tileSize

	totalTiles := gridWidth * gridHeight
	tileOrder := make([]int, totalTiles)
	for i := range totalTiles {
		tileOrder[i] = i
	}

	r := rand.New(rand.NewSource(42))
	r.Shuffle(len(tileOrder), func(i, j int) {
		tileOrder[i], tileOrder[j] = tileOrder[j], tileOrder[i]
	})

	return tileGrid{
		tileSize:   tileSize,
		gridWidth:  gridWidth,
		gridHeight: gridHeight,
		tileOrder:  tileOrder,
	}
}

func (g tileGrid) revealedTiles(progress float64) map[int]bool {
	totalTiles := len(g.tileOrder)
	revealedCount := int(math.Round(progress * float64(totalTiles)))

	revealedSet := make(map[int]bool)
	for i := range revealedCount {
		if i < len(g.tileOrder) {
			revealedSet[g.tileOrder[i]] = true
		}
	}
	return revealedSet
}

func (g tileGrid) tileIndex(x, y int) int {
	tileX := x / g.tileSize
	tileY := y / g.tileSize
	return tileY*g.gridWidth + tileX
}

type fade struct {
	width     int
	height    int
	fps       int
	spring    harmonica.Spring
	progress  float64
	vel       float64
	animating bool
	direction direction
	grid      tileGrid
}

func newFade(fps int) fade {
	const frequency = 15
	const damping = 0.65

	return fade{
		fps:    fps,
		spring: harmonica.NewSpring(harmonica.FPS(fps), frequency, damping),
	}
}

func (t fade) Start(width, height int, direction direction) Transition {
	t.width = width
	t.height = height
	t.animating = true
	t.progress = 0
	t.vel = 0
	t.direction = direction
	t.grid = newTileGrid(width, height, 2)

	return t
}

func (t fade) Animating() bool {
	return t.animating
}

func (t fade) Update() (Transition, tea.Cmd) {
	targetProgress := 1.0

	t.progress, t.vel = t.spring.Update(t.progress, t.vel, targetProgress)

	if t.progress >= 0.99 {
		t.animating = false
		t.progress = 1.0
		return t, nil
	}

	return t, Animate(time.Duration(t.fps))
}

func (t fade) View(prev, next string) string {
	var s strings.Builder

	prevLines := strings.Split(prev, "\n")
	nextLines := strings.Split(next, "\n")

	// Ensure slides are equal height
	maxLines := max(len(nextLines), len(prevLines))

	// Get revealed tiles from grid
	revealedSet := t.grid.revealedTiles(t.progress)
	allRevealed := len(revealedSet) >= len(t.grid.tileOrder)

	for lineIdx := range maxLines {
		var prevLine, nextLine string

		if lineIdx < len(prevLines) {
			prevLine = prevLines[lineIdx]
		}
		if lineIdx < len(nextLines) {
			nextLine = nextLines[lineIdx]
		}

		var line string
		if allRevealed {
			line = truncate.String(nextLine, uint(t.width))
		} else {
			line = t.buildFadeLine(prevLine, nextLine, revealedSet, lineIdx)
		}

		s.WriteString(line)
		if lineIdx < maxLines-1 {
			s.WriteString("\n")
		}
	}

	return s.String()
}

func (t fade) buildFadeLine(prevLine, nextLine string, revealedSet map[int]bool, lineIdx int) string {
	var result strings.Builder

	for tileX := range t.grid.gridWidth {
		startPos := tileX * t.grid.tileSize
		endPos := min(startPos+t.grid.tileSize, t.width)
		tileWidth := endPos - startPos

		tileIndex := t.grid.tileIndex(startPos, lineIdx)

		// Choose source line based on tile state
		sourceLine := prevLine
		if revealedSet[tileIndex] {
			sourceLine = nextLine
		}

		// Extract tile segment
		segment := t.extractTileSegment(sourceLine, startPos, tileWidth)
		result.WriteString(segment)
	}

	finalLine := result.String()
	if charmansi.StringWidth(finalLine) > t.width {
		finalLine = truncate.String(finalLine, uint(t.width))
	}

	return finalLine
}

func (t fade) extractTileSegment(line string, startPos, tileWidth int) string {
	if startPos == 0 {
		return truncate.String(line, uint(tileWidth))
	}

	skipped := skip.String(line, uint(startPos))
	return truncate.String(skipped, uint(tileWidth))
}

func (t fade) Name() string {
	return "fade"
}

func (t fade) Opposite() Transition {
	return newFade(t.fps)
}

func (t fade) Direction() direction {
	return t.direction
}
