package tui

import (
	"fmt"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/museslabs/kyma/internal/config"
	"github.com/museslabs/kyma/internal/markdown"
	"github.com/museslabs/kyma/internal/tui/transitions"
)

type Slide struct {
	Data             string
	Prev             *Slide
	Next             *Slide
	Style            config.SlideStyle
	ActiveTransition transitions.Transition
	Properties       config.Properties
	Timer            Timer

	renderer *markdown.Renderer

	// rendering is held while the renderer is in use, which it can also be
	// from the background, and mu while the renders kept below are used.
	rendering sync.Mutex
	mu        sync.Mutex
	// still and moving are the slide as last rendered on its own, as it
	// looks once a transition lands on it and as transitions draw it.
	still, moving rendered
}

// rendered is a slide drawn at a size.
type rendered struct {
	ok            bool
	width, height int
	view          string
}

type UpdateSlidesMsg struct {
	NewRoot *Slide
}

func NewSlide(data string, props config.Properties) (*Slide, error) {
	themeName := "dark"
	if props.Style.Theme != nil && props.Style.Theme.Name != "" {
		themeName = props.Style.Theme.Name
	}

	r, err := markdown.NewRenderer(themeName, markdown.WithImageBackend(props.ImageBackend))
	if err != nil {
		return nil, err

	}

	template, err := config.MasterLayout(props.Master)
	if err != nil {
		return nil, err
	}

	return &Slide{
		Data:       markdown.ApplyLayout(template, data),
		Properties: props,
		renderer:   r,
	}, nil
}

func (s *Slide) Update() (*Slide, tea.Cmd) {
	transition, cmd := s.ActiveTransition.Update()
	s.ActiveTransition = transition

	return s, cmd
}

func (s *Slide) View(animating bool, width, height int) string {
	still := s.render(
		s.Style,
		(s.ActiveTransition != nil && s.ActiveTransition.Animating()) || animating,
		width,
		height,
	)

	if s.ActiveTransition == nil || !s.ActiveTransition.Animating() {
		return still
	}

	if s.ActiveTransition.Direction() == transitions.Backwards {
		if s.Next == nil {
			panic("backwards transition at the last slide")
		}
		return s.ActiveTransition.View(s.Next.View(true, width, height), still)
	}
	if s.Prev != nil {
		return s.ActiveTransition.View(s.Prev.View(true, width, height), still)
	}
	return still
}

// render draws the slide on its own, inside its border and centered on a screen
// of width by height. Rendering markdown and
// images takes a while, and a transition draws both of its slides on every
// frame, so the slide is kept as rendered, still and for animating, for as long
// as the size stays the same.
func (s *Slide) render(style config.SlideStyle, animating bool, width, height int) string {
	if view, ok := s.kept(animating, width, height); ok {
		return view
	}

	// The renderer is shared with rendering in the background, and whichever
	// gets to it first renders for both.
	s.rendering.Lock()
	defer s.rendering.Unlock()
	if view, ok := s.kept(animating, width, height); ok {
		return view
	}

	// Grids need the space inside the slide's border and padding, not the
	// terminal's, or every column overflows by the width of the frame.
	frame := style.LipGlossStyle
	out, err := s.renderer.Render(
		s.Data,
		animating,
		width-frame.GetHorizontalFrameSize(),
		height-frame.GetVerticalFrameSize(),
	)
	if err != nil {
		out = fmt.Sprintf("# Error rendering slide\n\n%s", err)
	}

	// Centered on the screen here, which measures every line, image and all,
	// so that showing the slide doesn't have to.
	view := lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, frame.Render(out))

	s.mu.Lock()
	defer s.mu.Unlock()
	r := rendered{ok: true, width: width, height: height, view: view}
	if animating {
		s.moving = r
	} else {
		s.still = r
	}
	return view
}

// kept returns the render of the slide kept for a size, if there is one.
func (s *Slide) kept(animating bool, width, height int) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r := s.still
	if animating {
		r = s.moving
	}
	return r.view, r.ok && r.width == width && r.height == height
}

// Prerender renders the slide for a size ahead of time: first as transitions
// draw it and then as it looks once one lands on it, so that neither has to
// wait for its markdown and images when the time comes. It is safe to call
// from the background.
func (s *Slide) Prerender(width, height int) {
	if s.renderer == nil {
		return
	}
	style := s.Properties.Style.Apply(width, height)
	s.render(style, true, width, height)
	s.render(style, false, width, height)
}

func (s *Slide) First() *Slide {
	current := s
	for current.Prev != nil {
		current = current.Prev
	}
	return current
}

func (s *Slide) Last() *Slide {
	current := s
	for current.Next != nil {
		current = current.Next
	}
	return current
}
