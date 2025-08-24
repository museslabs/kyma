package markdown

import "testing"

func TestDump(t *testing.T) {
	tests := []struct {
		name string
		root Node
		want string
	}{
		{
			name: "simple",
			root: node(
				&MarkdownRootNode{},
				&GlamourNode{Text: "test"},
				&ImageNode{
					Label:  "img",
					Path:   "./image.png",
					Width:  100,
					Height: 50,
				},
				&GlamourNode{Text: "test2"},
			),
			want: `MarkdownRoot()
|-Glamour(Text: "test")
|-Image(Label: "img", Path: "./image.png", Width: 100, Height: 50)
└-Glamour(Text: "test2")`,
		},
		{
			name: "grid",
			root: node(
				&MarkdownRootNode{},
				&GlamourNode{Text: "test"},
				node(
					&GridNode{ColumnCount: 3},
					node(
						&GridColumnNode{Span: 1},
						&GlamourNode{Text: "Col1"},
						&GlamourNode{Text: "Col1a"},
					),
					node(
						&GridColumnNode{Span: 2},
						&GlamourNode{Text: "Col2"},
						&GlamourNode{Text: "Col2a"},
					),
				),
				&GlamourNode{Text: "test2"},
			),
			want: `MarkdownRoot()
|-Glamour(Text: "test")
|-Grid(ColumnCount: 3)
| |-GridColumn(Span: 1)
| | |-Glamour(Text: "Col1")
| | └-Glamour(Text: "Col1a")
| └-GridColumn(Span: 2)
|   |-Glamour(Text: "Col2")
|   └-Glamour(Text: "Col2a")
└-Glamour(Text: "test2")`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Dump(tt.root)
			if got != tt.want {
				t.Errorf("Dump() got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}

}
