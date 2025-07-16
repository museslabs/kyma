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
