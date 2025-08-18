package markdown

import (
	"bytes"
	"testing"
)

func TestCodeblockParser_Parse(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want Node
	}{
		{
			name: "Single code block node",
			in: []byte(
				"``c{1,3}\n#include <stdio.h>\n\nint main(void) {\n printf('a string');\nreturn 0;\n}\n```",
			),
			want: &CodeBlockNode{
				Language: "c",
				Code:     "#include <stdio.h>\n\nint main(void) {\n printf('a string');\nreturn 0;\n}",
				Ranges: []CodeBlockLineRange{
					{
						Start: 1,
						End:   1,
					},
					{
						Start: 3,
						End:   3,
					},
				},
				ShowLineNumbers: false,
				StartLine:       1,
			},
		},
		{
			name: "Line flags",
			in: []byte(
				"``c{1,3} --numbered --start-at-line 15\n#include <stdio.h>\n\nint main(void) {\n printf('a string');\nreturn 0;\n}\n```",
			),
			want: &CodeBlockNode{
				Language: "c",
				Code:     "#include <stdio.h>\n\nint main(void) {\n printf('a string');\nreturn 0;\n}",
				Ranges: []CodeBlockLineRange{
					{
						Start: 1,
						End:   1,
					},
					{
						Start: 3,
						End:   3,
					},
				},
				ShowLineNumbers: true,
				StartLine:       15,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewCodeBlockParser()

			got := p.Parse(bytes.NewReader(tt.in), nil)
			if Dump(got) != Dump(tt.want) {
				t.Errorf("Parse() got:\n%s\nwant:\n%s", Dump(got), Dump(tt.want))
			}
		})
	}
}

func TestCodeBlockParser_parseLines(t *testing.T) {
	tests := []struct {
		name    string
		in      []byte
		want    []CodeBlockLineRange
		wantErr bool
	}{
		{
			name:    "empty syntax",
			in:      []byte(""),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "single line",
			in:      []byte("{5}"),
			want:    []CodeBlockLineRange{{Start: 5, End: 5}},
			wantErr: false,
		},
		{
			name:    "single range",
			in:      []byte("{1-3}"),
			want:    []CodeBlockLineRange{{Start: 1, End: 3}},
			wantErr: false,
		},
		{
			name: "multiple lines",
			in:   []byte("{1,3,5}"),
			want: []CodeBlockLineRange{
				{Start: 1, End: 1},
				{Start: 3, End: 3},
				{Start: 5, End: 5},
			},
			wantErr: false,
		},
		{
			name:    "multiple ranges",
			in:      []byte("{1-3,7-9}"),
			want:    []CodeBlockLineRange{{Start: 1, End: 3}, {Start: 7, End: 9}},
			wantErr: false,
		},
		{
			name: "mixed lines and ranges",
			in:   []byte("{1,3-5,7}"),
			want: []CodeBlockLineRange{
				{Start: 1, End: 1},
				{Start: 3, End: 5},
				{Start: 7, End: 7},
			},
			wantErr: false,
		},
		{
			name:    "with spaces",
			in:      []byte("{ 1 - 3 , 5 }"),
			want:    []CodeBlockLineRange{{Start: 1, End: 3}, {Start: 5, End: 5}},
			wantErr: false,
		},
		{
			name:    "invalid range (start > end)",
			in:      []byte("{5-3}"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid syntax",
			in:      []byte("{abc}"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "without braces",
			in:      []byte("1-3,5"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "multiple range splits",
			in:      []byte("{1-3-5,5}"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "trailing dash",
			in:      []byte("{1-3,5-}"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "trailing comma",
			in:      []byte("{1-3,}"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "mixed characters",
			in:      []byte("{1-a3,4}"),
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewCodeBlockParser()
			r := bytes.NewReader(tt.in)
			_, _ = r.ReadByte() // burn the first {

			got, gotErr := p.parseLines(r)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("parseLines() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("parseLines() succeeded unexpectedly")
			}

			if len(got) != len(tt.want) {
				t.Errorf("parseLines() returned %d ranges, want %d", len(got), len(tt.want))
				return
			}

			for i, r := range got {
				if i >= len(tt.want) || r.Start != tt.want[i].Start ||
					r.End != tt.want[i].End {
					t.Errorf("parseLines() = %+v, want %+v", got, tt.want)
					break
				}
			}
		})
	}
}
