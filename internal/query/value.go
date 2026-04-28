package query

import (
	"fmt"
	"strings"
)

// ValueKind identifies the type of a runtime value.
type ValueKind int

const (
	ValNull ValueKind = iota
	ValBool
	ValNumber
	ValString
	ValHeading
	ValCode
	ValLink
	ValImage
	ValTable
	ValList
	ValArray
	ValObject
	ValDocument
)

func (k ValueKind) String() string {
	switch k {
	case ValNull:
		return "null"
	case ValBool:
		return "bool"
	case ValNumber:
		return "number"
	case ValString:
		return "string"
	case ValHeading:
		return "heading"
	case ValCode:
		return "code"
	case ValLink:
		return "link"
	case ValImage:
		return "image"
	case ValTable:
		return "table"
	case ValList:
		return "list"
	case ValArray:
		return "array"
	case ValObject:
		return "object"
	case ValDocument:
		return "document"
	}
	return "unknown"
}

// Value is a runtime value in the query language.
type Value struct {
	Kind ValueKind

	// Primitive values
	Bool   bool
	Number float64
	Str    string

	// Structured values
	Heading  *HeadingValue
	Code     *CodeValue
	Link     *LinkValue
	Image    *ImageValue
	Table    *TableValue
	List     *ListValue
	Array    []Value
	Object   map[string]Value
	Document *DocumentValue
}

var Null = Value{Kind: ValNull}

func BoolVal(b bool) Value     { return Value{Kind: ValBool, Bool: b} }
func NumberVal(n float64) Value { return Value{Kind: ValNumber, Number: n} }
func StringVal(s string) Value  { return Value{Kind: ValString, Str: s} }
func ArrayVal(a []Value) Value  { return Value{Kind: ValArray, Array: a} }

// HeadingValue holds a markdown heading.
type HeadingValue struct {
	Level   uint8
	Text    string
	Offset  int
	Line    int
	Content string // section content
	RawMd   string
	Index   int // position in headings list
}

// CodeValue holds a code block.
type CodeValue struct {
	Language  string
	Content   string
	StartLine int
	EndLine   int
}

// LinkValue holds a link.
type LinkValue struct {
	Text     string
	URL      string
	LinkType string // "external", "anchor", "relative", "wikilink"
	Offset   int
}

// ImageValue holds an image.
type ImageValue struct {
	Alt   string
	Src   string
	Title string
}

// TableValue holds a table.
type TableValue struct {
	Headers    []string
	Rows       [][]string
	Alignments []string
}

// ListValue holds a list.
type ListValue struct {
	Ordered bool
	Items   []ListItemValue
}

type ListItemValue struct {
	Content string
	Checked *bool
}

// DocumentValue holds document metadata.
type DocumentValue struct {
	Content      string
	HeadingCount int
	WordCount    int
}

// IsTruthy returns whether the value is considered truthy.
func (v Value) IsTruthy() bool {
	switch v.Kind {
	case ValNull:
		return false
	case ValBool:
		return v.Bool
	case ValNumber:
		return v.Number != 0
	case ValString:
		return v.Str != ""
	case ValArray:
		return len(v.Array) > 0
	default:
		return true
	}
}

// ToText returns a text representation of the value.
func (v Value) ToText() string {
	switch v.Kind {
	case ValNull:
		return "null"
	case ValBool:
		if v.Bool {
			return "true"
		}
		return "false"
	case ValNumber:
		if v.Number == float64(int64(v.Number)) {
			return fmt.Sprintf("%d", int64(v.Number))
		}
		return fmt.Sprintf("%g", v.Number)
	case ValString:
		return v.Str
	case ValHeading:
		if v.Heading != nil {
			return v.Heading.Text
		}
		return ""
	case ValCode:
		if v.Code != nil {
			return v.Code.Content
		}
		return ""
	case ValLink:
		if v.Link != nil {
			return v.Link.Text
		}
		return ""
	case ValImage:
		if v.Image != nil {
			return v.Image.Alt
		}
		return ""
	case ValTable:
		if v.Table != nil {
			return strings.Join(v.Table.Headers, " | ")
		}
		return ""
	case ValList:
		if v.List != nil {
			var items []string
			for _, i := range v.List.Items {
				items = append(items, i.Content)
			}
			return strings.Join(items, "\n")
		}
		return ""
	case ValDocument:
		if v.Document != nil {
			return v.Document.Content
		}
		return ""
	case ValArray:
		var parts []string
		for _, el := range v.Array {
			parts = append(parts, el.ToText())
		}
		return strings.Join(parts, "\n")
	case ValObject:
		var parts []string
		for k, val := range v.Object {
			parts = append(parts, fmt.Sprintf("%s: %s", k, val.ToText()))
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

// GetProperty returns a named property of the value, if it exists.
func (v Value) GetProperty(name string) (Value, bool) {
	switch v.Kind {
	case ValHeading:
		if v.Heading == nil {
			return Null, false
		}
		switch name {
		case "level":
			return NumberVal(float64(v.Heading.Level)), true
		case "text", "title":
			return StringVal(v.Heading.Text), true
		case "line":
			return NumberVal(float64(v.Heading.Line)), true
		case "content":
			return StringVal(v.Heading.Content), true
		case "md", "raw":
			return StringVal(v.Heading.RawMd), true
		}
	case ValCode:
		if v.Code == nil {
			return Null, false
		}
		switch name {
		case "lang", "language":
			return StringVal(v.Code.Language), true
		case "content", "text":
			return StringVal(v.Code.Content), true
		case "start_line":
			return NumberVal(float64(v.Code.StartLine)), true
		case "end_line":
			return NumberVal(float64(v.Code.EndLine)), true
		}
	case ValLink:
		if v.Link == nil {
			return Null, false
		}
		switch name {
		case "text":
			return StringVal(v.Link.Text), true
		case "url", "href":
			return StringVal(v.Link.URL), true
		case "type":
			return StringVal(v.Link.LinkType), true
		}
	case ValImage:
		if v.Image == nil {
			return Null, false
		}
		switch name {
		case "alt":
			return StringVal(v.Image.Alt), true
		case "src", "url":
			return StringVal(v.Image.Src), true
		case "title":
			return StringVal(v.Image.Title), true
		}
	case ValTable:
		if v.Table == nil {
			return Null, false
		}
		switch name {
		case "headers":
			arr := make([]Value, len(v.Table.Headers))
			for i, h := range v.Table.Headers {
				arr[i] = StringVal(h)
			}
			return ArrayVal(arr), true
		case "rows":
			rows := make([]Value, len(v.Table.Rows))
			for i, row := range v.Table.Rows {
				cells := make([]Value, len(row))
				for j, c := range row {
					cells[j] = StringVal(c)
				}
				rows[i] = ArrayVal(cells)
			}
			return ArrayVal(rows), true
		}
	case ValDocument:
		if v.Document == nil {
			return Null, false
		}
		switch name {
		case "heading_count":
			return NumberVal(float64(v.Document.HeadingCount)), true
		case "word_count":
			return NumberVal(float64(v.Document.WordCount)), true
		}
	case ValString:
		switch name {
		case "length", "len", "size":
			return NumberVal(float64(len([]rune(v.Str)))), true
		}
	case ValArray:
		switch name {
		case "length", "len", "size":
			return NumberVal(float64(len(v.Array))), true
		}
	}
	return Null, false
}
