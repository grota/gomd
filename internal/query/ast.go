// Package query provides a jq-like query language for navigating markdown documents.
package query

// Span represents a source location span.
type Span struct {
	Start int
	End   int
}

func newSpan(start, end int) Span { return Span{Start: start, End: end} }

func (s Span) merge(other Span) Span {
	if s.Start > other.Start {
		s.Start = other.Start
	}
	if s.End < other.End {
		s.End = other.End
	}
	return s
}

// Query is a complete query consisting of one or more comma-separated piped expressions.
type Query struct {
	Expressions []PipedExpr
}

// PipedExpr is expressions connected by pipes (|).
type PipedExpr struct {
	Stages []Expr
}

// Expr is a single expression.
type Expr interface {
	exprNode()
	exprSpan() Span
}

// base for all expression types
type baseExpr struct{ span Span }

func (b baseExpr) exprSpan() Span { return b.span }

// IdentityExpr is the "." identity expression.
type IdentityExpr struct{ baseExpr }

func (e *IdentityExpr) exprNode() {}

// ElementExpr is a document element selector like .h2, .code, .link.
type ElementExpr struct {
	baseExpr
	Kind    ElementKind
	Filters []Filter
	Index   *IndexOp
}

func (e *ElementExpr) exprNode() {}

// PropertyExpr accesses a property: .text, .level, .url, etc.
type PropertyExpr struct {
	baseExpr
	Name string
}

func (e *PropertyExpr) exprNode() {}

// FunctionExpr is a function call.
type FunctionExpr struct {
	baseExpr
	Name string
	Args []Expr
}

func (e *FunctionExpr) exprNode() {}

// ObjectExpr constructs an object: {key: expr}.
type ObjectExpr struct {
	baseExpr
	Pairs []ObjectPair
}

type ObjectPair struct {
	Key   string
	Value Expr
}

func (e *ObjectExpr) exprNode() {}

// ArrayExpr constructs an array: [expr, ...].
type ArrayExpr struct {
	baseExpr
	Elements []Expr
}

func (e *ArrayExpr) exprNode() {}

// ConditionalExpr is if...then...else...end.
type ConditionalExpr struct {
	baseExpr
	Condition  Expr
	ThenBranch Expr
	ElseBranch Expr // may be nil
}

func (e *ConditionalExpr) exprNode() {}

// HierarchyExpr is .h1 > .h2 or .h1 >> .code.
type HierarchyExpr struct {
	baseExpr
	Parent Expr
	Child  Expr
	Direct bool // true = >, false = >>
}

func (e *HierarchyExpr) exprNode() {}

// LiteralExpr is a literal value.
type LiteralExpr struct {
	baseExpr
	Value LiteralValue
}

func (e *LiteralExpr) exprNode() {}

// BinaryExpr is a binary operation.
type BinaryExpr struct {
	baseExpr
	Op    BinaryOp
	Left  Expr
	Right Expr
}

func (e *BinaryExpr) exprNode() {}

// UnaryExpr is a unary operation.
type UnaryExpr struct {
	baseExpr
	Op   UnaryOp
	Expr Expr
}

func (e *UnaryExpr) exprNode() {}

// GroupExpr is a parenthesized expression.
type GroupExpr struct {
	baseExpr
	Expr Expr
}

func (e *GroupExpr) exprNode() {}

// ElementKind is the type of document element.
type ElementKind int

const (
	ElemHeading ElementKind = iota
	ElemCode
	ElemLink
	ElemImage
	ElemTable
	ElemList
	ElemBlockquote
	ElemParagraph
	ElemFrontMatter
)

// HeadingLevel holds an optional heading level (0 = any).
type HeadingLevel struct {
	Level int // 0 = any, 1-6 = specific level
}

// ElementSpec holds the kind and optional heading level.
type ElementSpec struct {
	Kind         ElementKind
	HeadingLevel int // only for ElemHeading; 0 = any
}

// ParseElementKind parses an element kind from a string identifier.
func ParseElementKind(s string) (ElementSpec, bool) {
	switch s {
	case "h", "heading", "headings", "header", "headers":
		return ElementSpec{ElemHeading, 0}, true
	case "h1":
		return ElementSpec{ElemHeading, 1}, true
	case "h2":
		return ElementSpec{ElemHeading, 2}, true
	case "h3":
		return ElementSpec{ElemHeading, 3}, true
	case "h4":
		return ElementSpec{ElemHeading, 4}, true
	case "h5":
		return ElementSpec{ElemHeading, 5}, true
	case "h6":
		return ElementSpec{ElemHeading, 6}, true
	case "code", "codeblock", "codeblocks", "pre":
		return ElementSpec{ElemCode, 0}, true
	case "link", "links", "a", "anchor":
		return ElementSpec{ElemLink, 0}, true
	case "img", "image", "images":
		return ElementSpec{ElemImage, 0}, true
	case "table", "tables":
		return ElementSpec{ElemTable, 0}, true
	case "list", "lists", "ul", "ol":
		return ElementSpec{ElemList, 0}, true
	case "blockquote", "blockquotes", "quote", "quotes", "bq":
		return ElementSpec{ElemBlockquote, 0}, true
	case "para", "paragraph", "paragraphs", "p":
		return ElementSpec{ElemParagraph, 0}, true
	case "frontmatter", "fm", "meta", "yaml":
		return ElementSpec{ElemFrontMatter, 0}, true
	}
	return ElementSpec{}, false
}

// Filter is a filter applied to an element selector.
type Filter interface{ filterNode() }

// TextFilter filters by text content.
type TextFilter struct {
	Pattern string
	Exact   bool
	Span    Span
}

func (f *TextFilter) filterNode() {}

// RegexFilter filters by regular expression.
type RegexFilter struct {
	Pattern string
	Span    Span
}

func (f *RegexFilter) filterNode() {}

// TypeFilter filters by sub-type (e.g. link type, code language).
type TypeFilter struct {
	TypeName string
	Span     Span
}

func (f *TypeFilter) filterNode() {}

// IndexOp specifies element indexing or slicing.
type IndexOp struct {
	Kind  IndexKind
	Index int64        // for Single
	Start *int64       // for Slice
	End   *int64       // for Slice
}

type IndexKind int

const (
	IndexSingle  IndexKind = iota
	IndexSlice
	IndexIterate
)

// LiteralValue holds a compile-time constant.
type LiteralValue struct {
	Kind    LiteralKind
	Str     string
	Num     float64
	Bool    bool
}

type LiteralKind int

const (
	LitString LiteralKind = iota
	LitNumber
	LitBool
	LitNull
)

// BinaryOp enumerates binary operators.
type BinaryOp int

const (
	BinEq BinaryOp = iota
	BinNe
	BinLt
	BinLe
	BinGt
	BinGe
	BinAnd
	BinOr
	BinAdd
	BinSub
	BinMul
	BinDiv
	BinMod
	BinAlt // //
)

func (op BinaryOp) Precedence() int {
	switch op {
	case BinOr:
		return 1
	case BinAnd:
		return 2
	case BinEq, BinNe:
		return 3
	case BinLt, BinLe, BinGt, BinGe:
		return 4
	case BinAlt:
		return 5
	case BinAdd, BinSub:
		return 6
	case BinMul, BinDiv, BinMod:
		return 7
	}
	return 0
}

func (op BinaryOp) String() string {
	switch op {
	case BinEq:
		return "=="
	case BinNe:
		return "!="
	case BinLt:
		return "<"
	case BinLe:
		return "<="
	case BinGt:
		return ">"
	case BinGe:
		return ">="
	case BinAnd:
		return "and"
	case BinOr:
		return "or"
	case BinAdd:
		return "+"
	case BinSub:
		return "-"
	case BinMul:
		return "*"
	case BinDiv:
		return "/"
	case BinMod:
		return "%"
	case BinAlt:
		return "//"
	}
	return "?"
}

// UnaryOp enumerates unary operators.
type UnaryOp int

const (
	UnaryNot UnaryOp = iota
	UnaryNeg
)
