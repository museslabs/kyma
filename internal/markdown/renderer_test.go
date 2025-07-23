package markdown

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/x/exp/golden"
)

func TestRenderer_BasicRender(t *testing.T) {
	r, err := NewRenderer("dark")
	if err != nil {
		t.Fatalf("could not construct receiver type: %v", err)
	}

	got, gotErr := r.Render(
		`
Glamour
=======

A casual introduction. 你好世界!


## Let’s talk about artichokes

The _artichoke_ is mentioned as a garden plant in the 8th century BC by Homer
**and** Hesiod. The naturally occurring variant of the artichoke, the cardoon,
which is native to the Mediterranean area, also has records of use as a food
among the ancient Greeks and Romans. Pliny the Elder mentioned growing of
_carduus_ in Carthage and Cordoba.

> He holds him with a skinny hand,
> ‘There was a ship,’ quoth he.
> ‘Hold off! unhand me, grey-beard loon!’
> An artichoke, dropt he.

--Samuel Taylor Coleridge, [The Rime of the Ancient Mariner][rime]

[rime]: https://poetryfoundation.org/poems/43997/
		`, false)
	if gotErr != nil {
		t.Errorf("Render() failed: %v", gotErr)
	}

	golden.RequireEqual(t, got)
}

func TestRenderer_RenderCodeBlockWithHighlightedLines(t *testing.T) {
	r, err := NewRenderer("dark")
	if err != nil {
		t.Fatalf("could not construct receiver type: %v", err)
	}

	codeBlock := `package main

import "fmt"

func main() {
	fmt.Println("Hello World")
}
	`

	got, gotErr := r.Render(fmt.Sprintf("# Slide\n```go{3,6}\n%s\n```", codeBlock), false)
	if gotErr != nil {
		t.Errorf("Render() failed: %v", gotErr)
	}

	golden.RequireEqual(t, got)
}

func TestRenderer_RenderCodeBlockWithLineNumbers(t *testing.T) {
	r, err := NewRenderer("dark")
	if err != nil {
		t.Fatalf("could not construct receiver type: %v", err)
	}

	codeBlock := `package main

import "fmt"

func main() {
	fmt.Println("Hello World")
}`

	got, gotErr := r.Render(
		fmt.Sprintf("# Slide\n```go{3,6} --numbered\n%s\n```\n", codeBlock),
		false,
	)
	if gotErr != nil {
		t.Errorf("Render() failed: %v", gotErr)
	}

	golden.RequireEqual(t, got)
}

func TestRenderer_RenderCodeBlockWithStartFromLine(t *testing.T) {
	r, err := NewRenderer("dark")
	if err != nil {
		t.Fatalf("could not construct receiver type: %v", err)
	}

	codeBlock := `func (l *List[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		for e := l.head; e != nil; e = e.next {
			if !yield(e.val) {
				return
			}
		}
	}
}`

	got, gotErr := r.Render(
		fmt.Sprintf("# Slide\n```go{2-8} --numbered --start-at-line 10\n%s\n```\n", codeBlock),
		false,
	)
	if gotErr != nil {
		t.Errorf("Render() failed: %v", gotErr)
	}

	golden.RequireEqual(t, got)
}
