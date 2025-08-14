package markdown

import "testing"

func TestMarkdownParser_Parse(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want Node
	}{
		{
			name: "Basic single node",
			in:   []byte("# This is a string"),
			want: &MarkdownRootNode{children: []Node{
				&GlamourNode{Text: "# This is a string"},
			}},
		},
		{
			name: "Text followed by image",
			in:   []byte("# This is a string\n![alt text](./image.png)"),
			want: &MarkdownRootNode{
				children: []Node{
					&GlamourNode{Text: "# This is a string\n"},
					&ImageNode{Label: "alt text", Path: "./image.png"},
				},
			},
		},
		{
			name: "Image in between text",
			in:   []byte("# This is a string\n![alt text](./image.png)\n> Some other string"),
			want: &MarkdownRootNode{
				children: []Node{
					&GlamourNode{Text: "# This is a string\n"},
					&ImageNode{Label: "alt text", Path: "./image.png"},
					&GlamourNode{Text: "\n> Some other string"},
				},
			},
		},
		{
			name: "Try parse invalid image node",
			in:   []byte("# This is a string\n![not_an_image\n> Some other string"),
			want: &MarkdownRootNode{children: []Node{
				&GlamourNode{
					Text: "# This is a string\n![not_an_image\n> Some other string",
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewMarkdownParser()
			p.Register(Prioritized[Parser](NewImageParser(), 1))
			p.Register(Prioritized[Parser](NewCodeBlockParser(), 1))

			got := p.Parse(tt.in)
			if Dump(got) != Dump(tt.want) {
				t.Errorf("Parse() got:\n%s\nwant:\n%s", Dump(got), Dump(tt.want))
			}
		})
	}
}
