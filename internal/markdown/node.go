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
	Next() Node
	SetNext(Node)
}

func Dump(n Node) string {
	var b strings.Builder

	indent := 0
	for n != nil {
		b.WriteString(strings.ReplaceAll(n.String(), "\n", "\\n"))

		n = n.Next()
		if n != nil {
			b.WriteString("\n" + strings.Repeat("  ", indent) + "└-")
		}
		indent++
	}

	return b.String()
}

type GlamourNode struct {
	Text string

	next Node
}

func (n GlamourNode) Kind() NodeKind {
	return NodeKindGlamour
}

func (n GlamourNode) Next() Node {
	return n.next
}

func (n *GlamourNode) SetNext(node Node) {
	n.next = node
}

func (n GlamourNode) String() string {
	return fmt.Sprintf(`GlamourNode(Text: "%s")`, n.Text)
}

type ImageNode struct {
	Label  string
	Path   string
	Width  int
	Height int

	next Node
}

func (n ImageNode) Kind() NodeKind {
	return NodeKindImage
}

func (n ImageNode) Next() Node {
	return n.next
}

func (n *ImageNode) SetNext(node Node) {
	n.next = node
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

	next Node
}

func (n CodeBlockNode) Kind() NodeKind {
	return NodeKindCodeBlock
}

func (n CodeBlockNode) Next() Node {
	return n.next
}

func (n *CodeBlockNode) SetNext(node Node) {
	n.next = node
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
	ColumnNum int
	Columns   []GridColumnNode

	next Node
}

func (n GridNode) Kind() NodeKind {
	return NodeKindGrid
}

func (n GridNode) Next() Node {
	return n.next
}

func (n *GridNode) SetNext(node Node) {
	n.next = node
}

func (n GridNode) String() string {
	return fmt.Sprintf(`GridNode()`)
}

type GridColumnNode struct {
	Data []Node
	Span int
}

func (n GridColumnNode) Kind() NodeKind {
	return NodeKindGridColumn
}

func (n GridColumnNode) Next() Node {
	return nil
}

func (n GridColumnNode) SetNext(_ Node) {
	panic("")
}

func (n GridColumnNode) String() string {
	return fmt.Sprintf(`GridColumnNode()`)
}
