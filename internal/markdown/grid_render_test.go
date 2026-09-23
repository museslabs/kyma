package markdown

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// renderPlain renders in and returns it with the styling and the leading kitty
// reset stripped, which is the shape the layout assertions care about.
func renderPlain(t *testing.T, in string, width, height int) []string {
	t.Helper()

	r, err := NewRenderer("dark")
	if err != nil {
		t.Fatalf("could not construct renderer: %v", err)
	}

	out, err := r.Render(in, true, width, height)
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	out = strings.TrimPrefix(out, "\x1b_Ga=d\x1b\\")
	out = strings.TrimRight(out, "\n")

	lines := strings.Split(out, "\n")
	for i, line := range lines {
		lines[i] = ansi.Strip(line)
	}

	return lines
}

// TestRenderer_GridFitsItsWidth is the invariant that keeps columns from
// staircasing: every line a grid emits is exactly as wide as the space it was
// given, whatever it holds.
func TestRenderer_GridFitsItsWidth(t *testing.T) {
	cases := map[string]string{
		"plain columns": "[row]\n[col]left[/col]\n[col]right[/col]\n[/row]",
		"uneven spans":  "[row]\n[col span=3]wide[/col]\n[col]narrow[/col]\n[/row]",
		"borders and padding": "[row gap=2]\n" +
			"[col border=rounded pad=1]alpha[/col]\n" +
			"[col border=double pad=\"1 3\"]beta[/col]\n[/row]",
		"long words that cannot wrap": "[row]\n" +
			"[col]supercalifragilisticexpialidocious[/col]\n[col]x[/col]\n[/row]",
		"nested grid": "[grid]\n[col span=2]master[/col]\n" +
			"[col]\n[row]one[/row]\n[row]two[/row]\n[/col]\n[/grid]",
		"list and quote": "[row]\n[col]\n- a\n- b\n[/col]\n[col]\n> quoted\n[/col]\n[/row]",
	}

	for _, width := range []int{40, 63, 80} {
		for name, in := range cases {
			t.Run(name, func(t *testing.T) {
				for i, line := range renderPlain(t, in, width, 0) {
					if got := ansi.StringWidth(line); got != width {
						t.Errorf(
							"width %d: line %d is %d cells wide:\n%q",
							width, i, got, line,
						)
					}
				}
			})
		}
	}
}

func TestRenderer_GridColumnWidths(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		width int
		// want is the column each marker is expected to start at.
		want map[string]int
	}{
		{
			name:  "equal columns",
			in:    "[row]\n[col]AAA[/col]\n[col]BBB[/col]\n[/row]",
			width: 60,
			want:  map[string]int{"AAA": 2, "BBB": 32},
		},
		{
			name:  "span=2 takes two thirds",
			in:    "[row]\n[col span=2]AAA[/col]\n[col]BBB[/col]\n[/row]",
			width: 60,
			want:  map[string]int{"AAA": 2, "BBB": 42},
		},
		{
			name:  "an exact width is honoured",
			in:    "[row]\n[col width=20]AAA[/col]\n[col]BBB[/col]\n[/row]",
			width: 60,
			want:  map[string]int{"AAA": 2, "BBB": 22},
		},
		{
			name:  "a percentage is taken of the row",
			in:    "[row]\n[col width=25%]AAA[/col]\n[col]BBB[/col]\n[/row]",
			width: 80,
			want:  map[string]int{"AAA": 2, "BBB": 22},
		},
		{
			name:  "a gap pushes the next column along",
			in:    "[row gap=4]\n[col]AAA[/col]\n[col]BBB[/col]\n[/row]",
			width: 60,
			want:  map[string]int{"AAA": 2, "BBB": 34},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := renderPlain(t, tt.in, tt.width, 0)
			joined := strings.Join(lines, "\n")

			for marker, want := range tt.want {
				line := ""
				for _, l := range lines {
					if strings.Contains(l, marker) {
						line = l
						break
					}
				}
				if line == "" {
					t.Fatalf("marker %q missing from:\n%s", marker, joined)
				}
				if got := strings.Index(line, marker); got != want {
					t.Errorf("marker %q starts at column %d, want %d", marker, got, want)
				}
			}
		})
	}
}

// TestRenderer_GridKeepsContent guards against the silent content loss the
// first version of the grid was prone to.
func TestRenderer_GridKeepsContent(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "attributes do not swallow the grid",
			in:   "[grid]\n[column span=2]kept[/column]\n[column span=1]also kept[/column]\n[/grid]",
			want: []string{"kept", "also kept"},
		},
		{
			name: "loose text between tags survives",
			in:   "[grid]\nloose\n[col]in a column[/col]\n[/grid]",
			want: []string{"loose", "in a column"},
		},
		{
			name: "an unclosed tag keeps its content and says so",
			in:   "[grid]\n[col]\nstill here",
			want: []string{"still here", "unclosed [col]"},
		},
		{
			name: "every column renders, not just the first",
			in:   "[row]\n[col]one[/col]\n[col]\n## two\n\n- a\n- b\n[/col]\n[col]three[/col]\n[/row]",
			want: []string{"one", "two", "a", "b", "three"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := strings.Join(renderPlain(t, tt.in, 80, 0), "\n")
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Errorf("output is missing %q:\n%s", want, out)
				}
			}
		})
	}
}

// TestRenderer_GridHeight covers the rule that a grid is only as tall as its
// content unless a row asked for a share of the slide.
func TestRenderer_GridHeight(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		height int
		want   int
	}{
		{
			name:   "a plain row does not stretch",
			in:     "[row]\n[col]one[/col]\n[col]two[/col]\n[/row]",
			height: 24,
			want:   1,
		},
		{
			name:   "a row with a span fills the budget",
			in:     "[grid]\n[row span=1]one[/row]\n[/grid]",
			height: 10,
			want:   10,
		},
		{
			name:   "spans share the budget in proportion",
			in:     "[grid]\n[row span=3]top[/row]\n[row span=1]bottom[/row]\n[/grid]",
			height: 20,
			want:   20,
		},
		{
			name:   "an exact height is honoured",
			in:     "[grid]\n[row height=4]only[/row]\n[/grid]",
			height: 20,
			want:   4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := renderPlain(t, tt.in, 60, tt.height)
			if got := len(lines); got != tt.want {
				t.Errorf("rendered %d lines, want %d:\n%s", got, tt.want, strings.Join(lines, "\n"))
			}
		})
	}
}

func TestRenderer_GridAlignment(t *testing.T) {
	// A single centred column of width 60 puts a 3 cell marker at column 28.
	lines := renderPlain(t, "[row]\n[col align=center]XXX[/col]\n[/row]", 60, 0)

	for _, line := range lines {
		if i := strings.Index(line, "XXX"); i >= 0 {
			if i != 28 {
				t.Errorf("centred content starts at column %d, want 28", i)
			}
			return
		}
	}
	t.Fatalf("marker missing from:\n%s", strings.Join(lines, "\n"))
}

func TestRenderer_GridValign(t *testing.T) {
	tests := map[string]int{
		"top":    0,
		"middle": 4,
		"bottom": 9,
	}

	for valign, want := range tests {
		t.Run(valign, func(t *testing.T) {
			in := "[grid]\n[row span=1 height=10]\n[col valign=" + valign + "]XXX[/col]\n[/row]\n[/grid]"

			lines := renderPlain(t, in, 60, 10)
			for i, line := range lines {
				if strings.Contains(line, "XXX") {
					if i != want {
						t.Errorf("content is on line %d, want %d", i, want)
					}
					return
				}
			}
			t.Fatalf("marker missing from:\n%s", strings.Join(lines, "\n"))
		})
	}
}
