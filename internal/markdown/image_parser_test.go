package markdown

import (
	"bytes"
	"testing"
)

func TestImageParser_Parse(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want Node
	}{
		{
			name: "Single image node",
			in:   []byte("[alt text](./image.png)"),
			want: &ImageNode{
				Label: "alt text",
				Path:  "./image.png",
			},
		},
		{
			name: "Try parse invalid image node",
			in:   []byte("[not_an_image(path.png)"),
			want: nil,
		},
		{
			name: "Image node with custom width and height",
			in:   []byte("[alt text|20x10](./image.png)"),
			want: &ImageNode{
				Label:  "alt text",
				Path:   "./image.png",
				Width:  20,
				Height: 10,
			},
		},
		{
			name: "Image node with invalid width and height syntax",
			in:   []byte("[alt text|20,10](./image.png)"),
			want: nil,
		},
		{
			name: "Image node with invalid width and height params",
			in:   []byte("[alt text|2x1z](./image.png)"),
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewImageParser()

			got := p.Parse(bytes.NewReader(tt.in))
			if Dump(got) != Dump(tt.want) {
				t.Errorf("Parse() got:\n%s\nwant:\n%s", Dump(got), Dump(tt.want))
			}
		})
	}
}
