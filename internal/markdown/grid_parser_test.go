package markdown

import (
	"bytes"
	"testing"
)

func TestGridParser_Parse(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want Node
	}{
		{
			name: "simple grid",
			in:   []byte("[grid][column]Some text[/column][column]More text[/column][/grid]"),
			want: node(
				&GridNode{},
				node(&GridColumnNode{}, &GlamourNode{Text: "Some text"}),
				node(&GridColumnNode{}, &GlamourNode{Text: "More text"}),
			),
		},
		{
			name: "nested grid",
			in: []byte(`[grid]
[column]Some text[/column]

[column]
[grid]
[column]Nested text[/column]
[column]Nested text 2[/column]
[/grid]
[/column]
[/grid]`),
			want: node(
				&GridNode{},
				node(&GridColumnNode{}, &GlamourNode{Text: "Some text"}),
				node(&GridColumnNode{}, node(
					&GridNode{},
					node(&GridColumnNode{}, &GlamourNode{Text: "Nested text"}),
					node(&GridColumnNode{}, &GlamourNode{Text: "Nested text 2"}),
				)),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewGridParser()
			m := NewMarkdownParser()

			m.Register(Prioritized[Parser](NewGridParser(), 1))
			m.Register(Prioritized[Parser](NewGridColumnParser(), 2))

			got := p.Parse(bytes.NewReader(tt.in), m)
			if Dump(got) != Dump(tt.want) {
				t.Errorf("Parse() got:\n%s\nwant:\n%s", Dump(got), Dump(tt.want))
			}
		})
	}

}
