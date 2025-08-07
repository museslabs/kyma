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
			root: &GlamourNode{
				Text: "test",
				children: []Node{
					&ImageNode{
						Label:  "img",
						Path:   "./image.png",
						Width:  100,
						Height: 50,
						children: []Node{&GlamourNode{
							Text: "test2",
						}},
					},
				}},
			want: `GlamourNode(Text: "test")
└-ImageNode(Label: "img", Path: "./image.png", Width: 100, Height: 50)
  └-GlamourNode(Text: "test2")`,
		},
		{
			name: "grid",
			root: &GlamourNode{
				Text: "test",
				children: []Node{
					&GridNode{
						ColumnCount: 3,
						children: []Node{
							&GridColumnNode{
								Span: 1,
								children: []Node{
									&GlamourNode{
										Text:     "Col1",
										children: []Node{&GlamourNode{Text: "Col1Nest"}},
									},
									&GlamourNode{Text: "Col1a"},
								},
							},
							&GridColumnNode{
								Span: 2,
								children: []Node{
									&GlamourNode{Text: "Col2"},
									&GlamourNode{Text: "Col2a"},
								},
							},
							&GlamourNode{
								Text: "test2",
							},
						},
					}}},
			want: `GlamourNode(Text: "test")
└-GridNode(ColumnNum: 3)
  |-GridColumnNode(Span: 1)
  | |-GlamourNode(Text: "Col1")
  | | └-GlamourNode(Text: "Col1Nest")
  | └-GlamourNode(Text: "Col1a")
  |-GridColumnNode(Span: 2)
  | |-GlamourNode(Text: "Col2")
  | └-GlamourNode(Text: "Col2a")
  └-GlamourNode(Text: "test2")`,
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
