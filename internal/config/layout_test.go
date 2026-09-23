package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMasterLayout(t *testing.T) {
	dir := t.TempDir()

	named := filepath.Join(dir, "sidebar.md")
	if err := os.WriteFile(named, []byte("[row]\n[col][slot content][/col]\n[/row]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	withFrontMatter := filepath.Join(dir, "titled.md")
	body := "---\ntitle: ignored\n---\n[row]\n[col][slot content][/col]\n[/row]\n"
	if err := os.WriteFile(withFrontMatter, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	GlobalConfig.Masters = map[string]string{
		"two-col": "[row]\n[col][slot main][/col]\n[col][slot side][/col]\n[/row]",
	}
	t.Cleanup(func() { GlobalConfig.Masters = nil })

	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{
			name: "no layout",
			in:   "",
			want: "",
		},
		{
			name: "a name from the config file",
			in:   "two-col",
			want: GlobalConfig.Masters["two-col"],
		},
		{
			name: "a path to a markdown file",
			in:   named,
			want: "[row]\n[col][slot content][/col]\n[/row]\n",
		},
		{
			name: "a path without its extension",
			in:   strings.TrimSuffix(named, ".md"),
			want: "[row]\n[col][slot content][/col]\n[/row]\n",
		},
		{
			name: "front matter is stripped from a layout file",
			in:   withFrontMatter,
			want: "[row]\n[col][slot content][/col]\n[/row]\n",
		},
		{
			name:    "an unknown layout is an error, not a silently blank slide",
			in:      "does-not-exist",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MasterLayout(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("MasterLayout() error = %v, wantErr %t", err, tt.wantErr)
			}
			if got != tt.want && !tt.wantErr {
				t.Errorf("MasterLayout() = %q, want %q", got, tt.want)
			}
		})
	}
}
