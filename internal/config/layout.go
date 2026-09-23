package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MasterLayout resolves the master layout a slide asked for into the markdown
// template that frames its content. A name matches an entry under `masters:` in
// the config file first, and is otherwise read as a path to a markdown file,
// relative to the working directory. An empty name means the slide is not
// framed at all.
func MasterLayout(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}

	if template, ok := GlobalConfig.Masters[name]; ok {
		return template, nil
	}

	candidates := []string{name}
	if filepath.Ext(name) == "" {
		candidates = append(candidates, name+".md")
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		return stripFrontMatter(string(data)), nil
	}

	return "", fmt.Errorf(
		"master layout %q is not defined under `masters:` and no such file exists",
		name,
	)
}

// stripFrontMatter drops a leading `---` block so a layout can be kept in a
// markdown file that carries its own metadata.
func stripFrontMatter(s string) string {
	trimmed := strings.TrimLeft(s, " \t\n")
	if !strings.HasPrefix(trimmed, "---\n") {
		return s
	}

	_, rest, ok := strings.Cut(trimmed[len("---\n"):], "\n---")
	if !ok {
		return s
	}

	return strings.TrimPrefix(rest, "\n")
}
