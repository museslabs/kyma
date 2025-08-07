package markdown

import (
	"fmt"
	"strings"
)

type NodeKind uint8

const (
	NodeKindGlamour NodeKind = iota
	NodeKindImage
	NodeKindCodeBlock
	NodeKindGrid
	NodeKindGridColumn
)

type Node interface {
	fmt.Stringer

	Kind() NodeKind
	Children() []Node
	AddChild(Node)
}

func Dump(n Node) string {
	var b strings.Builder
	dumpNode(n, 0, &b)
	return b.String()
}

func dumpNode(n Node, indent int, b *strings.Builder) {
	if n == nil {
		return
	}

	b.WriteString(strings.ReplaceAll(n.String(), "\n", "\\n"))

	for _, c := range n.Children() {
		if c != nil {
			b.WriteString("\n" + strings.Repeat("  ", indent) + "└-")
			indent++
		}
		dumpNode(c, indent, b)
	}
}

type GlamourNode struct {
	Text string

	children []Node
}

func (n GlamourNode) Kind() NodeKind {
	return NodeKindGlamour
}

func (n GlamourNode) Children() []Node {
	return n.children
}

func (n *GlamourNode) AddChild(node Node) {
	n.children = append(n.children, node)
}

func (n GlamourNode) String() string {
	return fmt.Sprintf(`GlamourNode(Text: "%s")`, n.Text)
}

type ImageNode struct {
	Label  string
	Path   string
	Width  int
	Height int

	children []Node
}

func (n ImageNode) Kind() NodeKind {
	return NodeKindImage
}

func (n ImageNode) Children() []Node {
	return n.children
}

func (n *ImageNode) AddChild(node Node) {
	n.children = append(n.children, node)
}

func (n ImageNode) String() string {
	return fmt.Sprintf(
		`ImageNode(Label: "%s", Path: "%s", Width: %d, Height: %d)`,
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
	Language        string
	Ranges          []CodeBlockLineRange
	ShowLineNumbers bool
	StartLine       int
	Code            string

	children []Node
}

func (n CodeBlockNode) Kind() NodeKind {
	return NodeKindCodeBlock
}

func (n CodeBlockNode) Children() []Node {
	return n.children
}

func (n *CodeBlockNode) AddChild(node Node) {
	n.children = append(n.children, node)
}

func (n CodeBlockNode) String() string {
	return fmt.Sprintf(
		`CodeBlockNode(Language: %s, Ranges: %v, ShowLineNumbers: %t, StartLine: %d, Code: %s)`,
		n.Language,
		n.Ranges,
		n.ShowLineNumbers,
		n.StartLine,
		n.Code,
	)
}

type GridNode struct {
	ColumnCount int

	children []Node
}

func (n GridNode) Kind() NodeKind {
	return NodeKindGrid
}

func (n GridNode) Children() []Node {
	return n.children
}

func (n *GridNode) AddChild(node Node) {
	n.children = append(n.children, node)
}

func (n GridNode) String() string {
	return fmt.Sprintf(`GridNode(ColumnCount: %d)`, n.ColumnCount)
}

type GridColumnNode struct {
	Span int

	children []Node
}

func (n GridColumnNode) Kind() NodeKind {
	return NodeKindGridColumn
}

func (n GridColumnNode) Children() []Node {
	return n.children
}

func (n *GridColumnNode) AddChild(node Node) {
	n.children = append(n.children, node)
}

func (n GridColumnNode) String() string {
	return fmt.Sprintf(`GridColumnNode(Span: %d)`, n.Span)
}
