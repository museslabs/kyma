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
}

func parseHighlightSyntax(syntax string) []LineRange {
	if syntax == "" {
		return nil
	}

	syntax = strings.Trim(syntax, "{}")

	var ranges []LineRange
	parts := strings.Split(syntax, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			// Range like "1-3"
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) == 2 {
				start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
				end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
				if err1 == nil && err2 == nil && start <= end {
					ranges = append(ranges, LineRange{Start: start, End: end})
				}
			}
		} else {
			// Single line like "5"
			line, err := strconv.Atoi(part)
			if err == nil {
				ranges = append(ranges, LineRange{Start: line, End: line})
			}
		}
	}

	return ranges
}

func shouldHighlightLine(lineNum int, ranges []LineRange) bool {
	for _, r := range ranges {
		if lineNum >= r.Start && lineNum <= r.End {
			return true
		}
	}
	return false
}

func renderCustomCodeBlock(content string, info CodeHighlightInfo, themeName string) string {
	lexer := lexers.Get(info.Language)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := config.GetChromaStyle(themeName)
	return renderWithStyle(content, info, lexer, style)
}

func renderWithStyle(content string, info CodeHighlightInfo, lexer chroma.Lexer, style *chroma.Style) string {
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	lines := strings.Split(content, "\n")

	highlightStyle := lipgloss.NewStyle()

	lineNumberWidth := 0
	if info.ShowLineNumbers {
		lineNumberWidth = len(strconv.Itoa(len(lines))) + 2
	}

	lineNumberStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		PaddingRight(1)

	var result strings.Builder
	for lineNum, line := range lines {
		lineNumDisplay := lineNum + 1

		// Add line number if requested
		if info.ShowLineNumbers {
			lineNumStr := strconv.Itoa(lineNumDisplay)
			paddedLineNum := fmt.Sprintf("%*s", lineNumberWidth-1, lineNumStr)
			result.WriteString(lineNumberStyle.Render(paddedLineNum))
		}

		shouldHighlight := len(info.Ranges) == 0 || shouldHighlightLine(lineNumDisplay, info.Ranges)

		if !shouldHighlight {
			// No syntax highlighting for this line
			result.WriteString(line)
			if lineNum < len(lines)-1 {
				result.WriteString("\n")
			}
			continue
		}

		lineIterator, err := lexer.Tokenise(nil, line)
		if err != nil {
			result.WriteString(highlightStyle.Render(line))
			if lineNum < len(lines)-1 {
				result.WriteString("\n")
			}
			continue
		}

		var lineBuf strings.Builder
		err = formatter.Format(&lineBuf, style, lineIterator)
		if err != nil {
			result.WriteString(highlightStyle.Render(line))
			if lineNum < len(lines)-1 {
				result.WriteString("\n")
			}
			continue
		}

		syntaxHighlighted := lineBuf.String()
		syntaxHighlighted = strings.TrimRight(syntaxHighlighted, " \t\n\r")
		result.WriteString(highlightStyle.Render(syntaxHighlighted))

		if lineNum < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	codeStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Width(78).
		MarginTop(1).
		MarginBottom(1)

	return codeStyle.Render(result.String())
}

func processMarkdownWithHighlighting(markdown string, themeName string) (string, error) {
	re := regexp.MustCompile("(?s)```([a-zA-Z0-9_+-]*)({[^}]*})?(\\s+--numbered)?\\s*\n(.*?)\n```")

	matches := re.FindAllStringSubmatch(markdown, -1)
	if len(matches) == 0 {
		// No code blocks found, render the entire markdown with Glamour
		return glamour.Render(markdown, themeName)
	}

	var parts []string
	lastIndex := 0
	indices := re.FindAllStringIndex(markdown, -1)

	for i, match := range matches {
		if len(match) < 5 {
			continue
		}

		matchStart := indices[i][0]
		matchEnd := indices[i][1]

		// Add the text before this code block
		if matchStart > lastIndex {
			beforeText := markdown[lastIndex:matchStart]
			if strings.TrimSpace(beforeText) != "" {
				// Render regular markdown content with Glamour
				rendered, err := glamour.Render(beforeText, themeName)
				if err != nil {
					parts = append(parts, beforeText)
				} else {
					parts = append(parts, rendered)
				}
			}
		}

		// Extract code block info
		language := match[1] // e.g., "typescript"
		highlightSyntax := ""
		if len(match) > 2 && match[2] != "" {
			highlightSyntax = match[2] // e.g., "{1-2}"
		}
		numberedFlag := strings.TrimSpace(match[3]) // " --numbered"
		content := match[4]                         // The actual code content

		info := CodeHighlightInfo{
			Language:        language,
			Ranges:          parseHighlightSyntax(highlightSyntax),
			ShowLineNumbers: strings.Contains(numberedFlag, "--numbered"),
		}

		if language != "" || info.ShowLineNumbers {
			customRendered := renderCustomCodeBlock(content, info, themeName)
			parts = append(parts, customRendered)
		} else {
			// No language specified and no line numbers, use Glamour directly for the entire code block
			codeBlock := match[0]
			rendered, err := glamour.Render(codeBlock, themeName)
			if err != nil {
				parts = append(parts, codeBlock)
			} else {
				parts = append(parts, rendered)
			}
		}

		lastIndex = matchEnd
	}

	// Add any remaining text after the last code block
	if lastIndex < len(markdown) {
		remainingText := markdown[lastIndex:]
		if strings.TrimSpace(remainingText) != "" {
			// Render remaining markdown content with Glamour
			rendered, err := glamour.Render(remainingText, themeName)
			if err != nil {
				parts = append(parts, remainingText)
			} else {
				parts = append(parts, rendered)
			}
		}
	}

	return strings.Join(parts, ""), nil
}
