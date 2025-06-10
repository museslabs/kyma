package tui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/museslabs/kyma/internal/config"
)

type LineRange struct {
	Start int
	End   int
}

type CodeHighlightInfo struct {
	Language        string
	Ranges          []LineRange
	ShowLineNumbers bool
	StartLine       int
}

func parseHighlightSyntax(syntax string) []LineRange {
	if syntax == "" {
		return nil
	}

	syntax = strings.Trim(syntax, "{}")
	parts := strings.Split(syntax, ",")
	var ranges []LineRange

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			if r := parseRange(part); r != nil {
				ranges = append(ranges, *r)
			}
		} else {
			if line, err := strconv.Atoi(part); err == nil {
				ranges = append(ranges, LineRange{Start: line, End: line})
			}
		}
	}

	return ranges
}

func parseRange(part string) *LineRange {
	rangeParts := strings.Split(part, "-")
	if len(rangeParts) != 2 {
		return nil
	}

	start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
	end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))

	if err1 == nil && err2 == nil && start <= end {
		return &LineRange{Start: start, End: end}
	}
	return nil
}

func shouldHighlightLine(lineNum int, ranges []LineRange) bool {
	if len(ranges) == 0 {
		return true // highlight all lines if no ranges specified
	}

	for _, r := range ranges {
		if lineNum >= r.Start && lineNum <= r.End {
			return true
		}
	}
	return false
}

func formatLineNumber(lineNum, width int) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		PaddingRight(1)

	lineNumStr := strconv.Itoa(lineNum)
	paddedLineNum := fmt.Sprintf("%*s", width-1, lineNumStr)
	return style.Render(paddedLineNum)
}

func getLineNumberWidth(startLine, totalLines int) int {
	if totalLines == 0 {
		return 0
	}
	maxLineNum := startLine + totalLines - 1
	return len(strconv.Itoa(maxLineNum)) + 2
}

func renderPlainCode(lines []string, info CodeHighlightInfo) string {
	var result strings.Builder
	lineNumberWidth := 0

	if info.ShowLineNumbers {
		lineNumberWidth = getLineNumberWidth(info.StartLine, len(lines))
	}

	for i, line := range lines {
		displayLineNum := info.StartLine + i

		if info.ShowLineNumbers {
			result.WriteString(formatLineNumber(displayLineNum, lineNumberWidth))
		}

		result.WriteString(line)

		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

func renderHighlightedCode(content string, lines []string, info CodeHighlightInfo, lexer chroma.Lexer, style *chroma.Style) string {
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		return renderPlainCode(lines, info)
	}

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return renderPlainCode(lines, info)
	}

	var formattedBuf strings.Builder
	if err := formatter.Format(&formattedBuf, style, iterator); err != nil {
		return renderPlainCode(lines, info)
	}

	formattedLines := strings.Split(formattedBuf.String(), "\n")

	// Ensure we have the same number of lines
	for len(formattedLines) < len(lines) {
		formattedLines = append(formattedLines, "")
	}
	if len(formattedLines) > len(lines) {
		formattedLines = formattedLines[:len(lines)]
	}

	var result strings.Builder
	lineNumberWidth := 0

	if info.ShowLineNumbers {
		lineNumberWidth = getLineNumberWidth(info.StartLine, len(lines))
	}

	for i, line := range lines {
		displayLineNum := info.StartLine + i
		relativeLineNum := i + 1

		if info.ShowLineNumbers {
			result.WriteString(formatLineNumber(displayLineNum, lineNumberWidth))
		}

		if shouldHighlightLine(relativeLineNum, info.Ranges) {
			formattedLine := ""
			if i < len(formattedLines) {
				formattedLine = strings.TrimRight(formattedLines[i], " \t\n\r")
			}
			if formattedLine == "" {
				formattedLine = line
			}
			result.WriteString(formattedLine)
		} else {
			result.WriteString(line)
		}

		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

func renderCustomCodeBlock(content string, info CodeHighlightInfo, themeName string) string {
	lines := strings.Split(content, "\n")

	var renderedContent string

	if info.Language != "" {
		lexer := lexers.Get(info.Language)
		if lexer == nil {
			lexer = lexers.Fallback
		}
		lexer = chroma.Coalesce(lexer)
		style := config.GetChromaStyle(themeName)

		renderedContent = renderHighlightedCode(content, lines, info, lexer, style)
	} else {
		renderedContent = renderPlainCode(lines, info)
	}

	// Apply consistent styling
	codeStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Width(78).
		MarginTop(1).
		MarginBottom(1)

	return codeStyle.Render(renderedContent)
}

func parseCodeBlockInfo(match []string) CodeHighlightInfo {
	if len(match) < 7 {
		return CodeHighlightInfo{}
	}

	language := match[1]
	highlightSyntax := ""
	if len(match) > 2 && match[2] != "" {
		highlightSyntax = match[2]
	}
	numberedFlag := strings.TrimSpace(match[3])
	startLineStr := match[5]

	startLine := 1
	if startLineStr != "" {
		if parsed, err := strconv.Atoi(startLineStr); err == nil && parsed > 0 {
			startLine = parsed
		}
	}

	return CodeHighlightInfo{
		Language:        language,
		Ranges:          parseHighlightSyntax(highlightSyntax),
		ShowLineNumbers: strings.Contains(numberedFlag, "--numbered"),
		StartLine:       startLine,
	}
}

func renderMarkdownSection(text, themeName string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}

	rendered, err := glamour.Render(text, themeName)
	if err != nil {
		return text
	}
	return rendered
}

func processMarkdownWithHighlighting(markdown string, themeName string) (string, error) {
	// Regex to match code blocks with optional highlighting syntax
	re := regexp.MustCompile(`(?s)` + "`" + `{3}([a-zA-Z0-9_+-]*)({[^}]*})?(\s+--numbered)?(\s+--start-at-line\s+(\d+))?\s*\n(.*?)\n` + "`" + `{3}`)

	matches := re.FindAllStringSubmatch(markdown, -1)
	if len(matches) == 0 {
		return glamour.Render(markdown, themeName)
	}

	var parts []string
	lastIndex := 0
	indices := re.FindAllStringIndex(markdown, -1)

	for i, match := range matches {
		matchStart := indices[i][0]
		matchEnd := indices[i][1]

		// Add text before this code block
		if matchStart > lastIndex {
			beforeText := markdown[lastIndex:matchStart]
			parts = append(parts, renderMarkdownSection(beforeText, themeName))
		}

		// Process the code block
		info := parseCodeBlockInfo(match)
		content := match[6]

		if info.Language != "" || info.ShowLineNumbers {
			customRendered := renderCustomCodeBlock(content, info, themeName)
			parts = append(parts, customRendered)
		} else {
			// Use Glamour for plain code blocks
			codeBlock := match[0]
			parts = append(parts, renderMarkdownSection(codeBlock, themeName))
		}

		lastIndex = matchEnd
	}

	// Add remaining text after the last code block
	if lastIndex < len(markdown) {
		remainingText := markdown[lastIndex:]
		parts = append(parts, renderMarkdownSection(remainingText, themeName))
	}

	return strings.Join(parts, ""), nil
}
