package markdown

import (
	"fmt"
	"strings"
)

type NodeKind uint8

const (
	NodeKindMarkdownRoot NodeKind = iota
	NodeKindGlamour
	NodeKindImage
	NodeKindCodeBlock
	NodeKindGrid
	NodeKindGridRow
	NodeKindGridColumn
	NodeKindError
)

type Node interface {
	fmt.Stringer

	Kind() NodeKind
	Children() []Node
	AddChild(Node)
	Parent() Node
	SetParent(Node)
}

func Dump(n Node) string {
	var b strings.Builder
	dumpNode(n, "", &b)
	return b.String()
}

func dumpNode(n Node, prefix string, b *strings.Builder) {
	if n == nil {
		return
	}

	b.WriteString(strings.ReplaceAll(n.String(), "\n", "\\n"))

	for i, c := range n.Children() {
		if c == nil {
			continue
		}

		b.WriteString("\n" + prefix)

		if i == len(n.Children())-1 {
			b.WriteString("└-")
			dumpNode(c, prefix+"  ", b)
		} else {
			b.WriteString("|-")
			dumpNode(c, prefix+"| ", b)
		}
	}
}

type BaseNode struct {
	parent   Node
	children []Node
}

func (n BaseNode) Children() []Node {
	return n.children
}

func (n *BaseNode) AddChild(node Node) {
	n.children = append(n.children, node)
}

func (n BaseNode) Parent() Node {
	return n.parent
}

func (n *BaseNode) SetParent(node Node) {
	n.parent = node
}

type MarkdownRootNode struct {
	BaseNode
}

func (n MarkdownRootNode) Kind() NodeKind {
	return NodeKindMarkdownRoot
}

func (n MarkdownRootNode) String() string {
	return "MarkdownRoot()"
}

type GlamourNode struct {
	BaseNode

	Text string
}

func (n GlamourNode) Kind() NodeKind {
	return NodeKindGlamour
}

func (n GlamourNode) String() string {
	return fmt.Sprintf(`Glamour(Text: "%s")`, n.Text)
}

type ImageNode struct {
	BaseNode

	Label  string
	Path   string
	Width  int
	Height int
}

func (n ImageNode) Kind() NodeKind {
	return NodeKindImage
}

func (n ImageNode) String() string {
	return fmt.Sprintf(
		`Image(Label: "%s", Path: "%s", Width: %d, Height: %d)`,
		n.Label,
		n.Path,
		n.Width,
		n.Height,
	)
}

type CodeBlockLineRange struct {
	Start int
	End   int
}

type CodeBlockNode struct {
	BaseNode

	Language        string
	Ranges          []CodeBlockLineRange
	ShowLineNumbers bool
	StartLine       int
	Code            string
}

func (n CodeBlockNode) Kind() NodeKind {
	return NodeKindCodeBlock
}

func (n CodeBlockNode) String() string {
	return fmt.Sprintf(
		`CodeBlock(Language: %s, Ranges: %v, ShowLineNumbers: %t, StartLine: %d, Code: %s)`,
		n.Language,
		n.Ranges,
		n.ShowLineNumbers,
		n.StartLine,
		n.Code,
	)
}

// GridNode stacks its rows vertically. After parsing its children are always
// [GridRowNode]s: anything else written directly inside a [grid] is wrapped in
// an implicit row, see [normalizeGrid].
type GridNode struct {
	BaseNode

	Box
}

func (n GridNode) Kind() NodeKind {
	return NodeKindGrid
}

func (n GridNode) String() string {
	return fmt.Sprintf("Grid(%s)", n.attrString())
}

// GridRowNode lays its columns out side by side. After parsing its children are
// always [GridColumnNode]s, see [normalizeRow].
type GridRowNode struct {
	BaseNode

	Box
}

func (n GridRowNode) Kind() NodeKind {
	return NodeKindGridRow
}

func (n GridRowNode) String() string {
	return fmt.Sprintf("GridRow(%s)", n.attrString())
}

// GridColumnNode holds slide content, and may nest another [GridNode] to split
// its space further.
type GridColumnNode struct {
	BaseNode

	Box
}

func (n GridColumnNode) Kind() NodeKind {
	return NodeKindGridColumn
}

func (n GridColumnNode) String() string {
	return fmt.Sprintf("GridColumn(%s)", n.attrString())
}

// ErrorNode reports a malformed layout back to the author instead of silently
// dropping their content.
type ErrorNode struct {
	BaseNode

	Message string
}

func (n ErrorNode) Kind() NodeKind {
	return NodeKindError
}

func (n ErrorNode) String() string {
	return fmt.Sprintf(`Error(Message: "%s")`, n.Message)
}
