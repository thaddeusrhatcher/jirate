package renderer

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extAst "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
)

var _ renderer.Renderer = &ADFRenderer{}

// ADFRenderer implements goldmark.Renderer
type ADFRenderer struct {
	document *Node          // Root node
	context  blockNodeStack // Track where we are in the structure of the document
}

type Node struct {
	Type       NodeType     `json:"type"`
	Version    int          `json:"version,omitempty"`
	Attributes *Attributes  `json:"attrs,omitempty"`
	Content    []*Node      `json:"content,omitempty"`
	Marks      []MarkStruct `json:"marks,omitempty"`
	Text       string       `json:"text,omitempty"`
}

func (n *Node) AddContent(c *Node) {
	n.Content = append(n.Content, c)
}

type Attributes struct {
	Width    float32 `json:"width,omitempty"`    // For media single
	Layout   Layout  `json:"layout,omitempty"`   // For media single
	Level    int     `json:"level,omitempty"`    // For headings
	Language string  `json:"language,omitempty"` // For fenced code blocks
}

type MarkStruct struct {
	Type       Mark            `json:"type,omitempty"`
	Attributes *MarkAttributes `json:"attrs,omitempty"`
}

type MarkAttributes struct {
	Href  string `json:"href,omitempty"`  // For links
	Title string `json:"title,omitempty"` // For links
}

// Type represents the type of a node
type NodeType string

// Node types
const (
	NodeTypeNone        = "none"
	NodeTypeBlockquote  = "blockquote"
	NodeTypeBulletList  = "bulletList"
	NodeTypeCodeBlock   = "codeBlock"
	NodeTypeHeading     = "heading"
	NodeTypeMediaGroup  = "mediaGroup"
	NodeTypeMediaSingle = "mediaSingle"
	NodeTypeOrderedList = "orderedList"
	NodeTypePanel       = "panel"
	NodeTypeParagraph   = "paragraph"
	NodeTypeRule        = "rule"
	NodeTypeTable       = "table"
	NodeTypeListItem    = "listItem"
	NodeTypeMedia       = "media"
	NodeTypeTableCell   = "table_cell"
	NodeTypeTableHeader = "table_header"
	NodeTypeTableRow    = "table_row"
	NodeTypeEmoji       = "emoji"
	NodeTypeHardBreak   = "hardBreak"
	NodeTypeInlineCard  = "inlineCard"
	NodeTypeMention     = "mention"
	NodeTypeText        = "text"
)

func inlineType(t NodeType) bool {
	switch t {
	case NodeTypeEmoji, NodeTypeHardBreak, NodeTypeInlineCard, NodeTypeMention, NodeTypeText:
		return true
	default:
		return false
	}
}

type Layout string

// Enum values for Layout in Attributes struct
const (
	LayoutWrapLeft   = "wrap-left"
	LayoutCenter     = "center"
	LayoutWrapRight  = "wrap-right"
	LayoutWide       = "wide"
	LayoutFullWidth  = "full-width"
	LayoutAlignStart = "align-start"
	LayoutAlignEnd   = "align-end"
)

type blockNodeStack struct {
	data          []*Node
	ignoreBlocks  bool    // ADF does not support some forms of nesting that markdown does, so we sometimes ignore non-paragraph block nodes
	ignoredBlocks []*Node // Contains the root block and its children when a block node can only contain content
}

func (s *blockNodeStack) PushContent(node *Node) {
	s.PeekBlockNode().AddContent(node)
}

func (s *blockNodeStack) PushBlockNode(node *Node) {
	if s.ignoreBlocks {
		s.ignoredBlocks = append(s.ignoredBlocks, node)

		// Paragraphs are the only block node type that can still be added
		if node.Type != NodeTypeParagraph {
			return
		}
	}

	// Update the actual document
	s.PushContent(node)
	// Update the context stack
	s.data = append(s.data, node)
}

// Intentionally unsafe because we should never peek an empty stack
func (s *blockNodeStack) PeekBlockNode() *Node {
	return s.data[len(s.data)-1]
}

// Intentionally unsafe because we should never pop an empty stack
func (s *blockNodeStack) PopBlockNode() *Node {
	last := len(s.data) - 1
	node := s.data[last]

	if s.ignoreBlocks {
		s.ignoredBlocks = s.ignoredBlocks[:len(s.ignoredBlocks)-1]
		if len(s.ignoredBlocks) == 0 {
			s.ignoreBlocks = false
		} else if node.Type != NodeTypeParagraph {
			return node
		}
	}

	s.data = s.data[:last]
	return node
}

func (s *blockNodeStack) IgnoreNestedBlocks(node *Node) {
	if s.ignoreBlocks {
		return
	}

	s.ignoreBlocks = true
	s.ignoredBlocks = append(s.ignoredBlocks, node)
}

// Mark represents a text formatting directive
type Mark string

// Enum values for Mark text formatting
const (
	MarkCode      Mark = "code"
	MarkEm        Mark = "em"
	MarkLink      Mark = "link"
	MarkStrike    Mark = "strike"
	MarkStrong    Mark = "strong"
	MarkSubsup    Mark = "subsup"
	MarkTextcolor Mark = "textColor"
	MarkUnderline Mark = "underline"
)

func NewRenderer() *ADFRenderer {
	root := Node{
		Version: 1,
		Type:    "doc",
	}
	return &ADFRenderer{
		document: &root,
		context: blockNodeStack{
			data: []*Node{&root},
		},
	}
}

func Render(w io.Writer, source []byte) error {
	gm := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM, // GitHub flavoured markdown.
		),
		goldmark.WithParserOptions(
			parser.WithAttribute(), // Enables # headers {#custom-ids}.
		),
		goldmark.WithRenderer(NewRenderer()),
	)

	return gm.Convert(source, w)
}

func astToADFType(n ast.Node) NodeType {
	switch n.(type) {
	case *ast.Document:
	case *ast.Paragraph,
		*ast.TextBlock:
		return NodeTypeParagraph
	case *ast.Heading:
		return NodeTypeHeading
	case *ast.Text,
		*ast.String,
		*extAst.Strikethrough,
		*ast.Emphasis,
		*ast.CodeSpan,
		*ast.Link:
		return NodeTypeText
	case *ast.CodeBlock,
		*ast.FencedCodeBlock:
		return NodeTypeCodeBlock
	case *ast.ThematicBreak:
		return NodeTypeRule
	case *ast.Blockquote:
		return NodeTypeBlockquote
	case *ast.List:
		if n.(*ast.List).IsOrdered() {
			return NodeTypeOrderedList
		}
		return NodeTypeBulletList
	case *ast.ListItem:
		return NodeTypeListItem
	case *ast.Image:
		return NodeTypeMedia
	case *ast.HTMLBlock:
	case *ast.RawHTML:
	case *extAst.Table:
		return NodeTypeTable
	case *extAst.TableHeader:
		return NodeTypeTableHeader
	case *extAst.TableRow:
		return NodeTypeTableRow
	case *extAst.TableCell:
		return NodeTypeTableCell
	}

	return NodeTypeNone
}

func (r *ADFRenderer) walkNode(source []byte, n ast.Node, entering bool) ast.WalkStatus {
	// fmt.Printf("Node: %s, entering: %v, value: %q, children: %d\n", reflect.TypeOf(n).String(), entering, string(n.Text(source)), n.ChildCount())

	if !entering {
		if !inlineType(astToADFType(n)) {
			r.context.PopBlockNode()
		}
		return ast.WalkContinue
	}

	adfNode := &Node{Type: astToADFType(n)}

	switch ntype := n.(type) {
	case *ast.Document:
		// Nothing to do, the root ADF node is fixed.

	case *ast.Paragraph,
		*ast.TextBlock, // Untested
		*ast.List,
		*ast.ListItem,
		*ast.ThematicBreak,
		*ast.CodeBlock: // Untested
		r.context.PushBlockNode(adfNode)

	case *ast.Blockquote:
		r.context.PushBlockNode(adfNode)

		// ADF only supports paragraphs inside block quotes, no nested block quotes
		r.context.IgnoreNestedBlocks(adfNode)

	case *ast.Heading:
		adfNode.Attributes = &Attributes{
			Level: n.(*ast.Heading).Level,
		}
		r.context.PushBlockNode(adfNode)

	case *ast.Text,
		*ast.String: // Untested
		adfNode.Text = string(n.Text(source))
		if len(adfNode.Text) == 0 {
			// TODO: Uh what's happening here? Not sure why goldmark is splitting up paragraph text in this way.
			adfNode.Text = " "
		}
		r.context.PushContent(adfNode)

	case *ast.CodeSpan:
		adfNode.Text = string(n.Text(source))
		adfNode.Marks = []MarkStruct{{Type: MarkCode}}
		r.context.PushContent(adfNode)
		return ast.WalkSkipChildren

	case *extAst.Strikethrough:
		adfNode.Text = string(n.Text(source))
		adfNode.Marks = []MarkStruct{{Type: MarkStrike}}
		r.context.PushContent(adfNode)
		return ast.WalkSkipChildren

	case *ast.Emphasis:
		adfNode.Text = string(n.Text(source))
		if ntype.Level == 1 {
			adfNode.Marks = []MarkStruct{{Type: MarkEm}}
		} else if ntype.Level >= 2 {
			adfNode.Marks = []MarkStruct{{Type: MarkStrong}}
		}
		r.context.PushContent(adfNode)
		return ast.WalkSkipChildren

	case *ast.Link:
		adfNode.Text = string(n.Text(source))
		adfNode.Marks = []MarkStruct{{
			Type: MarkLink,
			Attributes: &MarkAttributes{
				Href:  string(ntype.Destination),
				Title: string(ntype.Title),
			},
		}}
		r.context.PushContent(adfNode)
		return ast.WalkSkipChildren

	case *ast.Image:
		// if entering {
		// 	children := r.renderChildren(source, n)
		// 	r.image(tnode.Destination, tnode.Title, children)
		// }
		// return ast.WalkSkipChildren

	case *ast.FencedCodeBlock:
		adfNode.Attributes = &Attributes{
			Language: string(ntype.Language(source)),
		}
		var content string
		lines := ntype.Lines()
		for i := 0; i < lines.Len(); i++ {
			segment := lines.At(i)
			content += string(segment.Value(source))
		}
		adfNode.AddContent(&Node{
			Type: NodeTypeText,
			Text: content,
		})
		r.context.PushBlockNode(adfNode)
		return ast.WalkSkipChildren

	case *ast.HTMLBlock:
		// if entering {
		// 	r.blockHtml(tnode, source)
		// }
	case *ast.RawHTML:
		// if entering {
		// 	r.rawHtml(tnode, source)
		// }
		// return ast.WalkSkipChildren
	case *extAst.Table:
		// r.table(tnode, entering)
	case *extAst.TableHeader:
		// if entering {
		// 	r.tableIsHeader = true
		// }
	case *extAst.TableRow:
		// if entering {
		// 	r.tableIsHeader = false
		// }
	case *extAst.TableCell:
		// if entering {
		// 	children := r.renderChildren(source, n)
		// 	if r.tableIsHeader {
		// 		r.tableHeaderCell(children, tnode.Alignment)
		// 	} else {
		// 		r.tableCell(children)
		// 	}
		// }
		// return ast.WalkSkipChildren
	default:
		panic("unknown type" + n.Kind().String())
	}

	return ast.WalkContinue
}

// Render implements goldmark.Renderer interface.
func (r *ADFRenderer) Render(w io.Writer, source []byte, n ast.Node) error {
	for current := n.FirstChild(); current != nil; current = current.NextSibling() {
		err := ast.Walk(current, func(current ast.Node, entering bool) (ast.WalkStatus, error) {
			return r.walkNode(source, current, entering), nil
		})
		if err != nil {
			return err
		}
	}

	b, err := json.MarshalIndent(r.document, "", "  ")
	if err != nil {
		return err
	}
	w.Write(b)

	return nil
}

func (*ADFRenderer) AddOptions(...renderer.Option) {
	//panic("No options for ADF renderer")
}
