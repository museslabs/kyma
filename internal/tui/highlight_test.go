package tui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

func TestParseHighlightSyntax(t *testing.T) {
	tests := []struct {
		name     string
		syntax   string
		expected []LineRange
	}{
		{
			name:     "empty syntax",
			syntax:   "",
			expected: nil,
		},
		{
			name:     "single line",
			syntax:   "{5}",
			expected: []LineRange{{Start: 5, End: 5}},
		},
		{
			name:     "single range",
			syntax:   "{1-3}",
			expected: []LineRange{{Start: 1, End: 3}},
		},
		{
			name:     "multiple lines",
			syntax:   "{1,3,5}",
			expected: []LineRange{{Start: 1, End: 1}, {Start: 3, End: 3}, {Start: 5, End: 5}},
		},
		{
			name:     "multiple ranges",
			syntax:   "{1-3,7-9}",
			expected: []LineRange{{Start: 1, End: 3}, {Start: 7, End: 9}},
		},
		{
			name:     "mixed lines and ranges",
			syntax:   "{1,3-5,7}",
			expected: []LineRange{{Start: 1, End: 1}, {Start: 3, End: 5}, {Start: 7, End: 7}},
		},
		{
			name:     "with spaces",
			syntax:   "{ 1 - 3 , 5 }",
			expected: []LineRange{{Start: 1, End: 3}, {Start: 5, End: 5}},
		},
		{
			name:     "invalid range (start > end)",
			syntax:   "{5-3}",
			expected: nil,
		},
		{
			name:     "invalid syntax",
			syntax:   "{abc}",
			expected: nil,
		},
		{
			name:     "without braces",
			syntax:   "1-3,5",
			expected: []LineRange{{Start: 1, End: 3}, {Start: 5, End: 5}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHighlightSyntax(tt.syntax)
			if len(result) != len(tt.expected) {
				t.Errorf("parseHighlightSyntax(%q) returned %d ranges, expected %d",
					tt.syntax, len(result), len(tt.expected))
				return
			}

			for i, r := range result {
				if i >= len(tt.expected) || r.Start != tt.expected[i].Start || r.End != tt.expected[i].End {
					t.Errorf("parseHighlightSyntax(%q) = %+v, expected %+v",
						tt.syntax, result, tt.expected)
					break
				}
			}
		})
	}
}

func TestShouldHighlightLine(t *testing.T) {
	ranges := []LineRange{
		{Start: 1, End: 3},
		{Start: 5, End: 5},
		{Start: 7, End: 10},
	}

	tests := []struct {
		lineNum  int
		expected bool
	}{
		{1, true},   // in first range
		{2, true},   // in first range
		{3, true},   // in first range
		{4, false},  // not in any range
		{5, true},   // in second range
		{6, false},  // not in any range
		{7, true},   // in third range
		{8, true},   // in third range
		{9, true},   // in third range
		{10, true},  // in third range
		{11, false}, // not in any range
		{0, false},  // edge case
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.lineNum)), func(t *testing.T) {
			result := shouldHighlightLine(tt.lineNum, ranges)
			if result != tt.expected {
				t.Errorf("shouldHighlightLine(%d, ranges) = %v, expected %v",
					tt.lineNum, result, tt.expected)
			}
		})
	}
}

func TestShouldHighlightLineEmptyRanges(t *testing.T) {
	var emptyRanges []LineRange

	result := shouldHighlightLine(5, emptyRanges)
	if result != true {
		t.Errorf("shouldHighlightLine(5, emptyRanges) = %v, expected true", result)
	}
}

func TestRenderCustomCodeBlock(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		info      CodeHighlightInfo
		themeName string
	}{
		{
			name:    "simple code with highlighting",
			content: "console.log('hello');\nconsole.log('world');",
			info: CodeHighlightInfo{
				Language: "javascript",
				Ranges:   []LineRange{{Start: 1, End: 1}},
			},
			themeName: "dark",
		},
		{
			name:    "python code with multiple ranges",
			content: "def hello():\n    print('hello')\n    print('world')\n    return True",
			info: CodeHighlightInfo{
				Language: "python",
				Ranges:   []LineRange{{Start: 1, End: 2}, {Start: 4, End: 4}},
			},
			themeName: "dark",
		},
		{
			name:    "unknown language fallback",
			content: "some random text\nmore text",
			info: CodeHighlightInfo{
				Language: "unknownlang",
				Ranges:   []LineRange{{Start: 1, End: 1}},
			},
			themeName: "dark",
		},
		{
			name:    "code with line numbers",
			content: "console.log('hello');\nconsole.log('world');\nvar x = 1;",
			info: CodeHighlightInfo{
				Language:        "javascript",
				Ranges:          []LineRange{{Start: 1, End: 1}},
				ShowLineNumbers: true,
			},
			themeName: "dark",
		},
		{
			name:    "code with only line numbers (no highlighting)",
			content: "def hello():\n    print('hello')\n    return True",
			info: CodeHighlightInfo{
				Language:        "python",
				Ranges:          nil,
				ShowLineNumbers: true,
			},
			themeName: "dark",
		},
		{
			name:    "code with start line offset",
			content: "console.log('hello');\nconsole.log('world');\nvar x = 1;",
			info: CodeHighlightInfo{
				Language:        "javascript",
				Ranges:          []LineRange{{Start: 2, End: 2}}, // Should highlight the second line
				ShowLineNumbers: true,
				StartLine:       10, // Line numbers should start at 10
			},
			themeName: "dark",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderCustomCodeBlock(tt.content, tt.info, tt.themeName)

			// Basic validation - should not be empty
			if result == "" {
				t.Error("renderCustomCodeBlock returned empty string")
			}

			// Result should have meaningful content (more than just whitespace)
			if len(strings.TrimSpace(result)) == 0 {
				t.Error("renderCustomCodeBlock returned only whitespace")
			}

			// Check for line numbers if requested
			if tt.info.ShowLineNumbers {
				if !strings.Contains(result, "1") && tt.info.StartLine == 1 {
					t.Error("renderCustomCodeBlock with ShowLineNumbers should contain line number 1")
				}
				if tt.info.StartLine > 1 {
					expectedFirstLine := strconv.Itoa(tt.info.StartLine)
					if !strings.Contains(result, expectedFirstLine) {
						t.Errorf("renderCustomCodeBlock with StartLine %d should contain line number %s", tt.info.StartLine, expectedFirstLine)
					}
				}
			}

			// For known languages, check that it contains some recognizable elements
			if tt.info.Language == "javascript" {
				if !strings.Contains(result, "console") && !strings.Contains(result, "log") {
					t.Error("renderCustomCodeBlock result doesn't contain expected javascript elements")
				}
			} else if tt.info.Language == "python" {
				if !strings.Contains(result, "def") && !strings.Contains(result, "print") {
					t.Error("renderCustomCodeBlock result doesn't contain expected python elements")
				}
			}
		})
	}
}

func TestRenderHighlightedCode(t *testing.T) {
	content := "console.log('test');\nvar x = 1;"
	lines := strings.Split(content, "\n")
	info := CodeHighlightInfo{
		Language: "javascript",
		Ranges:   []LineRange{{Start: 1, End: 1}},
	}

	lexer := lexers.Get("javascript")
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("github")
	if style == nil {
		style = styles.Fallback
	}

	result := renderHighlightedCode(content, lines, info, lexer, style)

	if result == "" {
		t.Error("renderHighlightedCode returned empty string")
	}

	// Result should have meaningful content (more than just whitespace)
	if len(strings.TrimSpace(result)) == 0 {
		t.Error("renderHighlightedCode returned only whitespace")
	}

	// Should contain some recognizable elements from the original content
	if !strings.Contains(result, "console") && !strings.Contains(result, "log") && !strings.Contains(result, "var") {
		t.Error("renderHighlightedCode result doesn't contain expected javascript elements")
	}
}

func TestRenderHighlightedCodeWithLineNumbers(t *testing.T) {
	content := "console.log('test');\nvar x = 1;\nfunction hello() {\n  return 'world';\n}"
	lines := strings.Split(content, "\n")
	info := CodeHighlightInfo{
		Language:        "javascript",
		Ranges:          []LineRange{{Start: 1, End: 2}},
		ShowLineNumbers: true,
	}

	lexer := lexers.Get("javascript")
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("github")
	if style == nil {
		style = styles.Fallback
	}

	result := renderHighlightedCode(content, lines, info, lexer, style)

	if result == "" {
		t.Error("renderHighlightedCode returned empty string")
	}

	// Should contain line numbers
	if !strings.Contains(result, "1") {
		t.Error("renderHighlightedCode with ShowLineNumbers should contain line number 1")
	}
	if !strings.Contains(result, "2") {
		t.Error("renderHighlightedCode with ShowLineNumbers should contain line number 2")
	}

	// Should contain some recognizable elements from the original content
	if !strings.Contains(result, "console") && !strings.Contains(result, "log") && !strings.Contains(result, "var") {
		t.Error("renderHighlightedCode result doesn't contain expected javascript elements")
	}
}

func TestProcessMarkdownWithHighlighting(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		theme    string
		wantErr  bool
	}{
		{
			name:     "markdown without code blocks",
			markdown: "# Hello\n\nThis is regular markdown text.",
			theme:    "dark",
			wantErr:  false,
		},
		{
			name:     "markdown with simple code block",
			markdown: "# Code Example\n\n```javascript\nconsole.log('hello');\n```",
			theme:    "dark",
			wantErr:  false,
		},
		{
			name:     "markdown with highlighted code block",
			markdown: "# Code Example\n\n```javascript{1}\nconsole.log('hello');\nconsole.log('world');\n```",
			theme:    "dark",
			wantErr:  false,
		},
		{
			name:     "markdown with multiple code blocks",
			markdown: "# Examples\n\n```javascript{1}\nconsole.log('hello');\n```\n\n```python{2}\nprint('hello')\nprint('world')\n```",
			theme:    "dark",
			wantErr:  false,
		},
		{
			name:     "markdown with range highlighting",
			markdown: "```python{1-2}\ndef hello():\n    print('hello')\n    return True\n```",
			theme:    "dark",
			wantErr:  false,
		},
		{
			name:     "markdown with mixed content",
			markdown: "# Title\n\nSome text before.\n\n```javascript{1}\nconsole.log('test');\n```\n\nSome text after.",
			theme:    "dark",
			wantErr:  false,
		},
		{
			name:     "markdown with --numbered flag",
			markdown: "```javascript --numbered\nconsole.log('hello');\nconsole.log('world');\n```",
			theme:    "dark",
			wantErr:  false,
		},
		{
			name:     "markdown with highlighting and --numbered",
			markdown: "```python{1-2} --numbered\ndef hello():\n    print('hello')\n    return True\n```",
			theme:    "dark",
			wantErr:  false,
		},
		{
			name:     "markdown with --numbered and extra spaces",
			markdown: "```javascript  --numbered  \nconsole.log('test');\nvar x = 1;\n```",
			theme:    "dark",
			wantErr:  false,
		},
		{
			name:     "markdown with --start-at-line flag",
			markdown: "```javascript --numbered --start-at-line 10\nconsole.log('hello');\nconsole.log('world');\n```",
			theme:    "dark",
			wantErr:  false,
		},
		{
			name:     "markdown with highlighting and --start-at-line",
			markdown: "```python{1-2} --numbered --start-at-line 5\ndef hello():\n    print('hello')\n    return True\n```",
			theme:    "dark",
			wantErr:  false,
		},
		{
			name:     "markdown with --start-at-line only (no --numbered)",
			markdown: "```typescript --start-at-line 100\nconst x = 1;\nconst y = 2;\n```",
			theme:    "dark",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processMarkdownWithHighlighting(tt.markdown, tt.theme)

			if (err != nil) != tt.wantErr {
				t.Errorf("processMarkdownWithHighlighting() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result == "" {
				t.Error("processMarkdownWithHighlighting returned empty string")
			}

			// For code blocks with highlighting, the result should be processed
			if strings.Contains(tt.markdown, "{") && strings.Contains(tt.markdown, "}") {
				// Should contain some processed content
				if len(result) < len(tt.markdown)/2 {
					t.Error("processMarkdownWithHighlighting result seems too short for highlighted content")
				}
			}

			// For code blocks with --numbered, should contain line numbers
			if strings.Contains(tt.markdown, "--numbered") {
				if !strings.Contains(result, "1") {
					t.Error("processMarkdownWithHighlighting with --numbered should contain line number 1")
				}
			}
		})
	}
}

func TestCodeHighlightInfo(t *testing.T) {
	info := CodeHighlightInfo{
		Language:        "javascript",
		Ranges:          []LineRange{{Start: 1, End: 3}, {Start: 5, End: 5}},
		ShowLineNumbers: true,
		StartLine:       10,
	}

	if info.Language != "javascript" {
		t.Errorf("Expected language 'javascript', got '%s'", info.Language)
	}

	if len(info.Ranges) != 2 {
		t.Errorf("Expected 2 ranges, got %d", len(info.Ranges))
	}

	if info.Ranges[0].Start != 1 || info.Ranges[0].End != 3 {
		t.Errorf("First range incorrect: got {%d, %d}, expected {1, 3}",
			info.Ranges[0].Start, info.Ranges[0].End)
	}

	if !info.ShowLineNumbers {
		t.Error("Expected ShowLineNumbers to be true")
	}

	if info.StartLine != 10 {
		t.Errorf("Expected StartLine to be 10, got %d", info.StartLine)
	}
}

func TestLineRange(t *testing.T) {
	lr := LineRange{Start: 1, End: 5}

	if lr.Start != 1 {
		t.Errorf("Expected Start = 1, got %d", lr.Start)
	}

	if lr.End != 5 {
		t.Errorf("Expected End = 5, got %d", lr.End)
	}
}
