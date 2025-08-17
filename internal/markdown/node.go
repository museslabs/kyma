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

type MarkdownRootNode struct {
	children []Node
}

func (n MarkdownRootNode) Kind() NodeKind {
	return NodeKindMarkdownRoot
}

func (n MarkdownRootNode) Children() []Node {
	return n.children
}

func (n *MarkdownRootNode) AddChild(node Node) {
	n.children = append(n.children, node)
}

func (n MarkdownRootNode) String() string {
	return "MarkdownRoot()"
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
	return fmt.Sprintf(`Glamour(Text: "%s")`, n.Text)
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
		`CodeBlock(Language: %s, Ranges: %v, ShowLineNumbers: %t, StartLine: %d, Code: %s)`,
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
	return fmt.Sprintf(`Grid(ColumnCount: %d)`, n.ColumnCount)
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
	return fmt.Sprintf(`GridColumn(Span: %d)`, n.Span)
}
