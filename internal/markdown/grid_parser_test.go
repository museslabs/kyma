package markdown

import "testing"

// gridParser builds a parser with the layout parsers registered, matching what
// [NewRenderer] sets up.
func gridParser() *MarkdownParser {
	m := NewMarkdownParser()
	m.Register(Prioritized[Parser](NewGridParser(), 1))
	m.Register(Prioritized[Parser](NewGridRowParser(), 2))
	m.Register(Prioritized[Parser](NewGridColumnParser(), 3))
	return m
}

func TestGridParser_Parse(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want Node
	}{
		{
			name: "columns are wrapped in an implicit row",
			in:   []byte("[grid][column]Some text[/column][column]More text[/column][/grid]"),
			want: node(
				&MarkdownRootNode{},
				node(
					&GridNode{},
					node(
						&GridRowNode{},
						node(&GridColumnNode{}, &GlamourNode{Text: "Some text"}),
						node(&GridColumnNode{}, &GlamourNode{Text: "More text"}),
					),
				)),
		},
		{
			name: "nested grid",
			in: []byte(`[grid]
[col]Some text[/col]

[col]
[grid]
[col]Nested text[/col]
[col]Nested text 2[/col]
[/grid]
[/col]
[/grid]`),
			want: node(
				&MarkdownRootNode{},
				node(
					&GridNode{},
					node(
						&GridRowNode{},
						node(&GridColumnNode{}, &GlamourNode{Text: "Some text"}),
						node(&GridColumnNode{}, node(
							&GridNode{},
							node(
								&GridRowNode{},
								node(&GridColumnNode{}, &GlamourNode{Text: "Nested text"}),
								node(&GridColumnNode{}, &GlamourNode{Text: "Nested text 2"}),
							),
						)),
					),
				)),
		},
		{
			name: "explicit rows stack, gap carries over",
			in: []byte(`[grid gap=2]
[row]
[col span=2]master[/col]
[col]stack[/col]
[/row]
[row height=3]footer[/row]
[/grid]`),
			want: node(
				&MarkdownRootNode{},
				node(
					&GridNode{Box: Box{Gap: 2}},
					node(
						&GridRowNode{Box: Box{Gap: 2}},
						node(
							&GridColumnNode{Box: Box{Size: Size{SizeFraction, 2}}},
							&GlamourNode{Text: "master"},
						),
						node(&GridColumnNode{}, &GlamourNode{Text: "stack"}),
					),
					node(
						&GridRowNode{Box: Box{Size: Size{SizeCells, 3}, Gap: 2}},
						node(&GridColumnNode{}, &GlamourNode{Text: "footer"}),
					),
				)),
		},
		{
			name: "attributes",
			in: []byte(`[grid]
[col width=30% align=center valign=middle border=rounded border_color="#ff0000" pad="1 2"]
sidebar
[/col]
[/grid]`),
			want: node(
				&MarkdownRootNode{},
				node(
					&GridNode{},
					node(
						&GridRowNode{},
						node(
							&GridColumnNode{Box: Box{
								Size:        Size{SizePercent, 30},
								Align:       AlignCenter,
								VAlign:      AlignCenter,
								PadX:        2,
								PadY:        1,
								Border:      "rounded",
								BorderColor: "#ff0000",
							}},
							&GlamourNode{Text: "sidebar"},
						),
					),
				)),
		},
		{
			name: "loose content is kept in an implicit column",
			in: []byte(`[grid]
stray text
[col]in a column[/col]
[/grid]`),
			want: node(
				&MarkdownRootNode{},
				node(
					&GridNode{},
					node(
						&GridRowNode{},
						node(&GridColumnNode{}, &GlamourNode{Text: "stray text"}),
					),
					node(
						&GridRowNode{},
						node(&GridColumnNode{}, &GlamourNode{Text: "in a column"}),
					),
				)),
		},
		{
			name: "a foreign closing tag closes the column implicitly",
			in: []byte(`[grid]
[col]
content that must survive
[/grid]`),
			want: node(
				&MarkdownRootNode{},
				node(
					&GridNode{},
					node(
						&GridRowNode{},
						node(
							&GridColumnNode{},
							&GlamourNode{Text: "content that must survive"},
						),
					),
				)),
		},
		{
			name: "running out of input reports the unclosed tag",
			in: []byte(`[grid]
[col]
content that must survive`),
			want: node(
				&MarkdownRootNode{},
				node(
					&GridNode{},
					node(
						&GridRowNode{},
						node(
							&GridColumnNode{},
							&GlamourNode{Text: "content that must survive"},
							&ErrorNode{Message: "unclosed [col]"},
						),
					),
					&ErrorNode{Message: "unclosed [grid]"},
				)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := gridParser()

			got := m.Parse(tt.in)
			if Dump(got) != Dump(tt.want) {
				t.Errorf("Parse() got:\n%s\nwant:\n%s", Dump(got), Dump(tt.want))
			}
		})
	}
}

// TestGridParser_LeavesMarkdownAlone guards the rule that an opening tag has to
// open its line, so ordinary links and bracketed text are never swallowed.
func TestGridParser_LeavesMarkdownAlone(t *testing.T) {
	tests := []string{
		"See [the docs](https://kyma.ink) for more",
		"A [grid] in the middle of a sentence",
		"[text][ref] reference style links",
	}

	for _, in := range tests {
		t.Run(in, func(t *testing.T) {
			m := gridParser()

			got := m.Parse([]byte(in))
			want := node(&MarkdownRootNode{}, &GlamourNode{Text: in})

			if Dump(got) != Dump(want) {
				t.Errorf("Parse() got:\n%s\nwant:\n%s", Dump(got), Dump(want))
			}
		})
	}
}

// TestGridParser_IgnoresCodeFences is what makes the syntax documentable: a
// layout tag inside a fenced code block is shown, not acted on.
func TestGridParser_IgnoresCodeFences(t *testing.T) {
	in := "Here is the syntax:\n\n```markdown\n[grid]\n[col span=2]\nmaster\n[/col]\n[/grid]\n```\n\nand that is it"

	got := gridParser().Parse([]byte(in))
	want := node(&MarkdownRootNode{}, &GlamourNode{Text: in})

	if Dump(got) != Dump(want) {
		t.Errorf("Parse() got:\n%s\nwant:\n%s", Dump(got), Dump(want))
	}
}

// TestGridParser_FenceInsideColumn covers the same rule one level down, where
// the fence is part of a column's content.
func TestGridParser_FenceInsideColumn(t *testing.T) {
	in := "[col]\n```markdown\n[row]not a row[/row]\n```\n[/col]"

	got := gridParser().Parse([]byte(in))
	want := node(
		&MarkdownRootNode{},
		node(
			&GridColumnNode{},
			&GlamourNode{Text: "```markdown\n[row]not a row[/row]\n```"},
		),
	)

	if Dump(got) != Dump(want) {
		t.Errorf("Parse() got:\n%s\nwant:\n%s", Dump(got), Dump(want))
	}
}

// TestGridParser_BareContainers covers using [row] or [col] on their own, which
// is the shortest way to split a slide.
func TestGridParser_BareContainers(t *testing.T) {
	m := gridParser()

	got := m.Parse([]byte("[row]\n[col]left[/col]\n[col]right[/col]\n[/row]"))
	want := node(
		&MarkdownRootNode{},
		node(
			&GridRowNode{},
			node(&GridColumnNode{}, &GlamourNode{Text: "left"}),
			node(&GridColumnNode{}, &GlamourNode{Text: "right"}),
		),
	)

	if Dump(got) != Dump(want) {
		t.Errorf("Parse() got:\n%s\nwant:\n%s", Dump(got), Dump(want))
	}
}
