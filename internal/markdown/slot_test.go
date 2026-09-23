package markdown

import (
	"maps"
	"testing"
)

func TestParseSlots(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want Slots
	}{
		{
			name: "a body with no slots is all default",
			in:   "# Title\n\nsome content",
			want: Slots{DefaultSlot: "# Title\n\nsome content"},
		},
		{
			name: "named blocks",
			in:   "[slot main]\n# Title\n[/slot]\n[slot side]\n- a\n- b\n[/slot]",
			want: Slots{"main": "# Title", "side": "- a\n- b"},
		},
		{
			name: "a slot on one line",
			in:   "[slot title]Hello[/slot]",
			want: Slots{"title": "Hello"},
		},
		{
			name: "content outside the slots lands in the default one",
			in:   "# Title\n[slot side]notes[/slot]\nmore body",
			want: Slots{DefaultSlot: "# Title\n\nmore body", "side": "notes"},
		},
		{
			name: "the same slot may be filled twice",
			in:   "[slot side]one[/slot]\n[slot side]two[/slot]",
			want: Slots{"side": "one\n\ntwo"},
		},
		{
			name: "an unnamed slot is the default one",
			in:   "[slot]\nbody\n[/slot]",
			want: Slots{DefaultSlot: "body"},
		},
		{
			name: "a slot tag inside a code fence is documentation",
			in:   "```markdown\n[slot side]\nnotes\n[/slot]\n```",
			want: Slots{DefaultSlot: "```markdown\n[slot side]\nnotes\n[/slot]\n```"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSlots(tt.in)
			if !maps.Equal(got, tt.want) {
				t.Errorf("ParseSlots() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestApplyLayout(t *testing.T) {
	const template = `[grid]
[col span=2]
[slot main]
[/col]
[col]
[slot side]
no notes yet
[/slot]
[/col]
[/grid]`

	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "slots are placed into the layout",
			body: "[slot main]\n# Title\n[/slot]\n[slot side]\n- a\n[/slot]",
			want: `[grid]
[col span=2]
# Title
[/col]
[col]
- a
[/col]
[/grid]`,
		},
		{
			name: "an unfilled slot keeps the layout's own default",
			body: "[slot main]\n# Title\n[/slot]",
			want: `[grid]
[col span=2]
# Title
[/col]
[col]
no notes yet
[/col]
[/grid]`,
		},
		{
			name: "a body with no slot tags fills the default slot",
			body: "# Just a title",
			want: `[grid]
[col span=2]
[/col]
[col]
no notes yet
[/col]
[/grid]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ApplyLayout(template, tt.body); got != tt.want {
				t.Errorf("ApplyLayout() =\n%s\n\nwant:\n%s", got, tt.want)
			}
		})
	}
}

// TestApplyLayout_DefaultSlot covers the shorthand where a layout names its one
// hole "content" and the slide just writes markdown.
func TestApplyLayout_DefaultSlot(t *testing.T) {
	template := "[row]\n[col]\n[slot content]\n[/col]\n[col]\n[slot aside]\n[/col]\n[/row]"

	got := ApplyLayout(template, "# Title\n\nbody text")
	want := "[row]\n[col]\n# Title\n\nbody text\n[/col]\n[col]\n[/col]\n[/row]"

	if got != want {
		t.Errorf("ApplyLayout() =\n%q\n\nwant:\n%q", got, want)
	}
}

// TestApplyLayout_NoLayout is the fallback for a slide that names slots but has
// no layout to fill: the content stays, the tags go.
func TestApplyLayout_NoLayout(t *testing.T) {
	got := ApplyLayout("", "[slot main]\n# Title\n[/slot]\nloose\n[slot side]notes[/slot]")
	want := "# Title\nloose\nnotes"

	if got != want {
		t.Errorf("ApplyLayout() = %q, want %q", got, want)
	}
}
