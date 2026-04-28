package query

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	mdparser "github.com/grota/gomd/internal/parser"
)

// EvalError is a query evaluation error.
type EvalError struct {
	Message string
}

func (e *EvalError) Error() string { return e.Message }

// EvalContext holds the evaluation state.
type EvalContext struct {
	Current   Value
	Headings  []HeadingValue
	CodeBlocks []CodeValue
	Links     []LinkValue
	Images    []ImageValue
	Tables    []TableValue
	Lists     []ListValue
	Document  DocumentValue
}

func contextFromDocument(doc *mdparser.Document) EvalContext {
	headings := extractHeadings(doc)
	codeBlocks := extractCodeBlocks(doc)
	links := extractLinks(doc)
	images := extractImages(doc)
	tables := extractTables(doc)
	lists := extractLists(doc)

	docVal := DocumentValue{
		Content:      doc.Content,
		HeadingCount: len(doc.Headings),
		WordCount:    len(strings.Fields(doc.Content)),
	}

	return EvalContext{
		Current:    Value{Kind: ValDocument, Document: &docVal},
		Headings:   headings,
		CodeBlocks: codeBlocks,
		Links:      links,
		Images:     images,
		Tables:     tables,
		Lists:      lists,
		Document:   docVal,
	}
}

// Engine executes queries against a document.
type Engine struct {
	doc *mdparser.Document
	ctx EvalContext
}

// NewEngine creates a new evaluation engine.
func NewEngine(doc *mdparser.Document) *Engine {
	return &Engine{doc: doc, ctx: contextFromDocument(doc)}
}

// Execute runs the query and returns results.
func (e *Engine) Execute(q *Query) ([]Value, error) {
	var all []Value
	for _, piped := range q.Expressions {
		results, err := e.evalPiped(&piped)
		if err != nil {
			return nil, err
		}
		all = append(all, results...)
	}
	return all, nil
}

func (e *Engine) evalPiped(piped *PipedExpr) ([]Value, error) {
	current := []Value{{Kind: ValDocument, Document: &e.ctx.Document}}
	for _, stage := range piped.Stages {
		var next []Value
		for _, input := range current {
			e.ctx.Current = input
			results, err := e.evalExpr(stage)
			if err != nil {
				return nil, err
			}
			next = append(next, results...)
		}
		current = next
		if len(current) == 0 {
			break
		}
	}
	return current, nil
}

func (e *Engine) evalExpr(expr Expr) ([]Value, error) {
	switch ex := expr.(type) {
	case *IdentityExpr:
		return []Value{e.ctx.Current}, nil

	case *ElementExpr:
		return e.evalElement(ex.Kind, 0, ex.Filters, ex.Index)

	case *headingElementExpr:
		return e.evalElement(ElemHeading, ex.HeadingLevel, ex.ElementExpr.Filters, ex.ElementExpr.Index)

	case *PropertyExpr:
		return e.evalProperty(ex.Name, ex.span)

	case *FunctionExpr:
		return e.evalFunction(ex.Name, ex.Args, ex.span)

	case *HierarchyExpr:
		return e.evalHierarchy(ex)

	case *BinaryExpr:
		return e.evalBinary(ex)

	case *UnaryExpr:
		return e.evalUnary(ex)

	case *LiteralExpr:
		return []Value{literalToValue(ex.Value)}, nil

	case *ObjectExpr:
		return e.evalObject(ex)

	case *ArrayExpr:
		return e.evalArray(ex)

	case *ConditionalExpr:
		return e.evalConditional(ex)

	case *GroupExpr:
		return e.evalExpr(ex.Expr)
	}
	return nil, &EvalError{fmt.Sprintf("unknown expression type %T", expr)}
}

func (e *Engine) evalElement(kind ElementKind, headingLevel int, filters []Filter, index *IndexOp) ([]Value, error) {
	var elements []Value

	switch kind {
	case ElemHeading:
		for i, h := range e.ctx.Headings {
			hCopy := e.ctx.Headings[i]
			if headingLevel == 0 || int(h.Level) == headingLevel {
				elements = append(elements, Value{Kind: ValHeading, Heading: &hCopy})
			}
		}
	case ElemCode:
		for i := range e.ctx.CodeBlocks {
			c := e.ctx.CodeBlocks[i]
			elements = append(elements, Value{Kind: ValCode, Code: &c})
		}
	case ElemLink:
		for i := range e.ctx.Links {
			l := e.ctx.Links[i]
			elements = append(elements, Value{Kind: ValLink, Link: &l})
		}
	case ElemImage:
		for i := range e.ctx.Images {
			img := e.ctx.Images[i]
			elements = append(elements, Value{Kind: ValImage, Image: &img})
		}
	case ElemTable:
		for i := range e.ctx.Tables {
			t := e.ctx.Tables[i]
			elements = append(elements, Value{Kind: ValTable, Table: &t})
		}
	case ElemList:
		for i := range e.ctx.Lists {
			l := e.ctx.Lists[i]
			elements = append(elements, Value{Kind: ValList, List: &l})
		}
	}

	// Apply filters
	var err error
	for _, f := range filters {
		elements, err = applyFilter(elements, f)
		if err != nil {
			return nil, err
		}
	}

	// Apply index
	if index != nil {
		elements = applyIndex(elements, index)
	}

	return elements, nil
}

func applyFilter(elements []Value, f Filter) ([]Value, error) {
	switch ft := f.(type) {
	case *TextFilter:
		pattern := strings.ToLower(ft.Pattern)
		var result []Value
		for _, v := range elements {
			text := strings.ToLower(v.ToText())
			if ft.Exact {
				if text == pattern {
					result = append(result, v)
				}
			} else {
				if strings.Contains(text, pattern) {
					result = append(result, v)
				}
			}
		}
		return result, nil
	case *RegexFilter:
		re, err := regexp.Compile(ft.Pattern)
		if err != nil {
			return nil, &EvalError{fmt.Sprintf("invalid regex %q: %v", ft.Pattern, err)}
		}
		var result []Value
		for _, v := range elements {
			if re.MatchString(v.ToText()) {
				result = append(result, v)
			}
		}
		return result, nil
	case *TypeFilter:
		typeName := strings.ToLower(ft.TypeName)
		var result []Value
		for _, v := range elements {
			switch v.Kind {
			case ValLink:
				if v.Link != nil && strings.ToLower(v.Link.LinkType) == typeName {
					result = append(result, v)
				}
			case ValCode:
				if v.Code != nil && strings.ToLower(v.Code.Language) == typeName {
					result = append(result, v)
				}
			}
		}
		return result, nil
	}
	return elements, nil
}

func applyIndex(elements []Value, idx *IndexOp) []Value {
	n := len(elements)
	switch idx.Kind {
	case IndexSingle:
		i := int(idx.Index)
		if i < 0 {
			i = n + i
		}
		if i >= 0 && i < n {
			return []Value{elements[i]}
		}
		return nil
	case IndexSlice:
		start := 0
		end := n
		if idx.Start != nil {
			s := int(*idx.Start)
			if s < 0 {
				s = n + s
			}
			if s < 0 {
				s = 0
			}
			start = s
		}
		if idx.End != nil {
			en := int(*idx.End)
			if en < 0 {
				en = n + en
			}
			if en > n {
				en = n
			}
			end = en
		}
		if start < end && start < n {
			return elements[start:end]
		}
		return nil
	case IndexIterate:
		return elements
	}
	return elements
}

func (e *Engine) evalProperty(name string, span Span) ([]Value, error) {
	v, ok := e.ctx.Current.GetProperty(name)
	if !ok {
		return nil, &EvalError{fmt.Sprintf("property %q not found on %s", name, e.ctx.Current.Kind)}
	}
	return []Value{v}, nil
}

func (e *Engine) evalFunction(name string, args []Expr, span Span) ([]Value, error) {
	// Handle special internal pipe
	if name == "_pipe" {
		current := []Value{e.ctx.Current}
		for _, arg := range args {
			var next []Value
			for _, inp := range current {
				e.ctx.Current = inp
				results, err := e.evalExpr(arg)
				if err != nil {
					return nil, err
				}
				next = append(next, results...)
			}
			current = next
		}
		return current, nil
	}

	return callBuiltin(name, args, e)
}

func (e *Engine) evalHierarchy(ex *HierarchyExpr) ([]Value, error) {
	parentVals, err := e.evalExpr(ex.Parent)
	if err != nil {
		return nil, err
	}

	var results []Value
	for _, pv := range parentVals {
		if pv.Kind != ValHeading || pv.Heading == nil {
			continue
		}
		parentIdx := pv.Heading.Index
		parentLevel := int(pv.Heading.Level)

		// Determine child element kind
		var childKind ElementKind
		var childHeadingLevel int
		switch cv := ex.Child.(type) {
		case *ElementExpr:
			childKind = cv.Kind
		case *headingElementExpr:
			childKind = ElemHeading
			childHeadingLevel = cv.HeadingLevel
		default:
			continue
		}

		if childKind == ElemHeading {
			for i := parentIdx + 1; i < len(e.ctx.Headings); i++ {
				h := &e.ctx.Headings[i]
				if int(h.Level) <= parentLevel {
					break
				}
				if childHeadingLevel != 0 && int(h.Level) != childHeadingLevel {
					continue
				}
				if ex.Direct {
					// Only include immediate children (no intermediate heading between parent and child)
					hasIntermediate := false
					for j := parentIdx + 1; j < i; j++ {
						intermediate := &e.ctx.Headings[j]
						if int(intermediate.Level) > parentLevel && int(intermediate.Level) < int(h.Level) {
							hasIntermediate = true
							break
						}
					}
					if hasIntermediate {
						continue
					}
				}
				hCopy := *h
				results = append(results, Value{Kind: ValHeading, Heading: &hCopy})
			}
		} else if childKind == ElemCode {
			// Simplified: return all code blocks (scoping by heading would require content offsets)
			for i := range e.ctx.CodeBlocks {
				c := e.ctx.CodeBlocks[i]
				results = append(results, Value{Kind: ValCode, Code: &c})
			}
		}

		// Apply child filters/index
		switch cv := ex.Child.(type) {
		case *ElementExpr:
			for _, f := range cv.Filters {
				results, err = applyFilter(results, f)
				if err != nil {
					return nil, err
				}
			}
			if cv.Index != nil {
				results = applyIndex(results, cv.Index)
			}
		case *headingElementExpr:
			for _, f := range cv.ElementExpr.Filters {
				results, err = applyFilter(results, f)
				if err != nil {
					return nil, err
				}
			}
			if cv.ElementExpr.Index != nil {
				results = applyIndex(results, cv.ElementExpr.Index)
			}
		}
	}

	return results, nil
}

func (e *Engine) evalBinary(ex *BinaryExpr) ([]Value, error) {
	leftVals, err := e.evalExpr(ex.Left)
	if err != nil {
		return nil, err
	}
	rightVals, err := e.evalExpr(ex.Right)
	if err != nil {
		return nil, err
	}

	lv := Null
	if len(leftVals) > 0 {
		lv = leftVals[0]
	}
	rv := Null
	if len(rightVals) > 0 {
		rv = rightVals[0]
	}

	var result Value
	switch ex.Op {
	case BinEq:
		result = BoolVal(valuesEqual(lv, rv))
	case BinNe:
		result = BoolVal(!valuesEqual(lv, rv))
	case BinLt:
		result = BoolVal(compareValues(lv, rv) < 0)
	case BinLe:
		result = BoolVal(compareValues(lv, rv) <= 0)
	case BinGt:
		result = BoolVal(compareValues(lv, rv) > 0)
	case BinGe:
		result = BoolVal(compareValues(lv, rv) >= 0)
	case BinAnd:
		result = BoolVal(lv.IsTruthy() && rv.IsTruthy())
	case BinOr:
		result = BoolVal(lv.IsTruthy() || rv.IsTruthy())
	case BinAdd:
		result = addValues(lv, rv)
	case BinSub:
		result = subValues(lv, rv)
	case BinMul:
		result = mulValues(lv, rv)
	case BinDiv:
		r, err2 := divValues(lv, rv)
		if err2 != nil {
			return nil, err2
		}
		result = r
	case BinMod:
		r, err2 := modValues(lv, rv)
		if err2 != nil {
			return nil, err2
		}
		result = r
	case BinAlt:
		if lv.IsTruthy() {
			result = lv
		} else {
			result = rv
		}
	}

	return []Value{result}, nil
}

func (e *Engine) evalUnary(ex *UnaryExpr) ([]Value, error) {
	vals, err := e.evalExpr(ex.Expr)
	if err != nil {
		return nil, err
	}
	v := Null
	if len(vals) > 0 {
		v = vals[0]
	}
	switch ex.Op {
	case UnaryNot:
		return []Value{BoolVal(!v.IsTruthy())}, nil
	case UnaryNeg:
		if v.Kind == ValNumber {
			return []Value{NumberVal(-v.Number)}, nil
		}
		return []Value{Null}, nil
	}
	return []Value{Null}, nil
}

func (e *Engine) evalObject(ex *ObjectExpr) ([]Value, error) {
	obj := make(map[string]Value)
	for _, pair := range ex.Pairs {
		vals, err := e.evalExpr(pair.Value)
		if err != nil {
			return nil, err
		}
		if len(vals) == 1 {
			obj[pair.Key] = vals[0]
		} else {
			obj[pair.Key] = ArrayVal(vals)
		}
	}
	return []Value{{Kind: ValObject, Object: obj}}, nil
}

func (e *Engine) evalArray(ex *ArrayExpr) ([]Value, error) {
	var arr []Value
	for _, elem := range ex.Elements {
		vals, err := e.evalExpr(elem)
		if err != nil {
			return nil, err
		}
		arr = append(arr, vals...)
	}
	return []Value{ArrayVal(arr)}, nil
}

func (e *Engine) evalConditional(ex *ConditionalExpr) ([]Value, error) {
	conds, err := e.evalExpr(ex.Condition)
	if err != nil {
		return nil, err
	}
	cond := Null
	if len(conds) > 0 {
		cond = conds[0]
	}
	if cond.IsTruthy() {
		return e.evalExpr(ex.ThenBranch)
	}
	if ex.ElseBranch != nil {
		return e.evalExpr(ex.ElseBranch)
	}
	return []Value{Null}, nil
}

// Helper math/compare functions

func valuesEqual(a, b Value) bool {
	if a.Kind != b.Kind {
		return a.ToText() == b.ToText()
	}
	switch a.Kind {
	case ValNull:
		return true
	case ValBool:
		return a.Bool == b.Bool
	case ValNumber:
		return math.Abs(a.Number-b.Number) < 1e-10
	case ValString:
		return a.Str == b.Str
	}
	return a.ToText() == b.ToText()
}

func compareValues(a, b Value) int {
	switch {
	case a.Kind == ValNumber && b.Kind == ValNumber:
		if a.Number < b.Number {
			return -1
		} else if a.Number > b.Number {
			return 1
		}
		return 0
	default:
		return strings.Compare(a.ToText(), b.ToText())
	}
}

func addValues(a, b Value) Value {
	switch {
	case a.Kind == ValNumber && b.Kind == ValNumber:
		return NumberVal(a.Number + b.Number)
	case a.Kind == ValString && b.Kind == ValString:
		return StringVal(a.Str + b.Str)
	case a.Kind == ValArray && b.Kind == ValArray:
		combined := make([]Value, len(a.Array)+len(b.Array))
		copy(combined, a.Array)
		copy(combined[len(a.Array):], b.Array)
		return ArrayVal(combined)
	default:
		return StringVal(a.ToText() + b.ToText())
	}
}

func subValues(a, b Value) Value {
	if a.Kind == ValNumber && b.Kind == ValNumber {
		return NumberVal(a.Number - b.Number)
	}
	return Null
}

func mulValues(a, b Value) Value {
	if a.Kind == ValNumber && b.Kind == ValNumber {
		return NumberVal(a.Number * b.Number)
	}
	if a.Kind == ValString && b.Kind == ValNumber {
		return StringVal(strings.Repeat(a.Str, int(b.Number)))
	}
	if a.Kind == ValNumber && b.Kind == ValString {
		return StringVal(strings.Repeat(b.Str, int(a.Number)))
	}
	return Null
}

func divValues(a, b Value) (Value, error) {
	if a.Kind == ValNumber && b.Kind == ValNumber {
		if b.Number == 0 {
			return Null, &EvalError{"division by zero"}
		}
		return NumberVal(a.Number / b.Number), nil
	}
	return Null, nil
}

func modValues(a, b Value) (Value, error) {
	if a.Kind == ValNumber && b.Kind == ValNumber {
		if b.Number == 0 {
			return Null, &EvalError{"division by zero"}
		}
		return NumberVal(math.Mod(a.Number, b.Number)), nil
	}
	return Null, nil
}

// Extraction helpers

func extractHeadings(doc *mdparser.Document) []HeadingValue {
	headings := make([]HeadingValue, len(doc.Headings))
	for i, h := range doc.Headings {
		// Calculate content end
		contentStart := h.Offset
		if nl := strings.Index(doc.Content[h.Offset:], "\n"); nl >= 0 {
			contentStart = h.Offset + nl + 1
		}
		contentEnd := len(doc.Content)
		for j := i + 1; j < len(doc.Headings); j++ {
			if doc.Headings[j].Level <= h.Level {
				contentEnd = doc.Headings[j].Offset
				break
			}
		}
		content := strings.TrimSpace(doc.Content[contentStart:contentEnd])
		rawMd := doc.Content[h.Offset:contentEnd]

		headings[i] = HeadingValue{
			Level:   uint8(h.Level),
			Text:    h.Text,
			Offset:  h.Offset,
			Line:    h.Line,
			Content: content,
			RawMd:   rawMd,
			Index:   i,
		}
	}
	return headings
}

func extractCodeBlocks(doc *mdparser.Document) []CodeValue {
	blocks := mdparser.ExtractCodeBlocks(doc.Content)
	result := make([]CodeValue, len(blocks))
	for i, b := range blocks {
		result[i] = CodeValue{
			Language:  b.Language,
			Content:   b.Content,
			StartLine: b.StartLine,
			EndLine:   b.EndLine,
		}
	}
	return result
}

func extractLinks(doc *mdparser.Document) []LinkValue {
	links := mdparser.ExtractLinks(doc.Content)
	result := make([]LinkValue, len(links))
	for i, l := range links {
		result[i] = LinkValue{
			Text:     l.Text,
			URL:      l.URL,
			LinkType: l.Type.String(),
			Offset:   l.Offset,
		}
	}
	return result
}

func extractImages(doc *mdparser.Document) []ImageValue {
	images := mdparser.ExtractImages(doc.Content)
	result := make([]ImageValue, len(images))
	for i, img := range images {
		result[i] = ImageValue{
			Alt:   img.Alt,
			Src:   img.Src,
			Title: img.Title,
		}
	}
	return result
}

func extractTables(doc *mdparser.Document) []TableValue {
	tables := mdparser.ExtractTables(doc.Content)
	result := make([]TableValue, len(tables))
	for i, t := range tables {
		result[i] = TableValue{
			Headers:    t.Headers,
			Rows:       t.Rows,
			Alignments: t.Alignments,
		}
	}
	return result
}

func extractLists(doc *mdparser.Document) []ListValue {
	lists := mdparser.ExtractLists(doc.Content)
	result := make([]ListValue, len(lists))
	for i, l := range lists {
		items := make([]ListItemValue, len(l.Items))
		for j, item := range l.Items {
			items[j] = ListItemValue{Content: item.Content, Checked: item.Checked}
		}
		result[i] = ListValue{Ordered: l.Ordered, Items: items}
	}
	return result
}

func literalToValue(lit LiteralValue) Value {
	switch lit.Kind {
	case LitString:
		return StringVal(lit.Str)
	case LitNumber:
		return NumberVal(lit.Num)
	case LitBool:
		return BoolVal(lit.Bool)
	case LitNull:
		return Null
	}
	return Null
}

// callBuiltin dispatches to built-in functions.
func callBuiltin(name string, args []Expr, e *Engine) ([]Value, error) {
	current := e.ctx.Current

	switch name {
	// --- Property shortcuts ---
	case "text", "title":
		return []Value{StringVal(current.ToText())}, nil
	case "level":
		if current.Kind == ValHeading && current.Heading != nil {
			return []Value{NumberVal(float64(current.Heading.Level))}, nil
		}
		return []Value{Null}, nil
	case "url", "href":
		if current.Kind == ValLink && current.Link != nil {
			return []Value{StringVal(current.Link.URL)}, nil
		}
		if current.Kind == ValImage && current.Image != nil {
			return []Value{StringVal(current.Image.Src)}, nil
		}
		return []Value{Null}, nil
	case "src":
		if current.Kind == ValImage && current.Image != nil {
			return []Value{StringVal(current.Image.Src)}, nil
		}
		return []Value{Null}, nil
	case "lang", "language":
		if current.Kind == ValCode && current.Code != nil {
			return []Value{StringVal(current.Code.Language)}, nil
		}
		return []Value{Null}, nil
	case "content":
		if current.Kind == ValHeading && current.Heading != nil {
			return []Value{StringVal(current.Heading.Content)}, nil
		}
		return []Value{StringVal(current.ToText())}, nil
	case "md", "raw":
		if current.Kind == ValHeading && current.Heading != nil {
			return []Value{StringVal(current.Heading.RawMd)}, nil
		}
		return []Value{StringVal(current.ToText())}, nil

	// --- Collection functions ---
	case "count", "length", "len", "size":
		if current.Kind == ValArray {
			return []Value{NumberVal(float64(len(current.Array)))}, nil
		}
		return []Value{NumberVal(1)}, nil

	case "first", "head":
		if current.Kind == ValArray {
			if len(current.Array) > 0 {
				return []Value{current.Array[0]}, nil
			}
			return []Value{Null}, nil
		}
		return []Value{current}, nil

	case "last":
		if current.Kind == ValArray {
			if len(current.Array) > 0 {
				return []Value{current.Array[len(current.Array)-1]}, nil
			}
			return []Value{Null}, nil
		}
		return []Value{current}, nil

	case "reverse":
		if current.Kind == ValArray {
			rev := make([]Value, len(current.Array))
			for i, v := range current.Array {
				rev[len(current.Array)-1-i] = v
			}
			return []Value{ArrayVal(rev)}, nil
		}
		return []Value{current}, nil

	case "sort":
		if current.Kind == ValArray {
			sorted := make([]Value, len(current.Array))
			copy(sorted, current.Array)
			sort.Slice(sorted, func(i, j int) bool {
				return compareValues(sorted[i], sorted[j]) < 0
			})
			return []Value{ArrayVal(sorted)}, nil
		}
		return []Value{current}, nil

	case "unique":
		if current.Kind == ValArray {
			seen := make(map[string]bool)
			var unique []Value
			for _, v := range current.Array {
				key := v.ToText()
				if !seen[key] {
					seen[key] = true
					unique = append(unique, v)
				}
			}
			return []Value{ArrayVal(unique)}, nil
		}
		return []Value{current}, nil

	case "flatten":
		if current.Kind == ValArray {
			var flat []Value
			for _, v := range current.Array {
				if v.Kind == ValArray {
					flat = append(flat, v.Array...)
				} else {
					flat = append(flat, v)
				}
			}
			return []Value{ArrayVal(flat)}, nil
		}
		return []Value{current}, nil

	case "limit", "take":
		if len(args) == 0 {
			return []Value{current}, nil
		}
		nVals, err := e.evalExpr(args[0])
		if err != nil {
			return nil, err
		}
		n := 0
		if len(nVals) > 0 && nVals[0].Kind == ValNumber {
			n = int(nVals[0].Number)
		}
		if current.Kind == ValArray {
			if n > len(current.Array) {
				n = len(current.Array)
			}
			return []Value{ArrayVal(current.Array[:n])}, nil
		}
		return []Value{current}, nil

	case "skip", "drop":
		if len(args) == 0 {
			return []Value{current}, nil
		}
		nVals, err := e.evalExpr(args[0])
		if err != nil {
			return nil, err
		}
		n := 0
		if len(nVals) > 0 && nVals[0].Kind == ValNumber {
			n = int(nVals[0].Number)
		}
		if current.Kind == ValArray {
			if n > len(current.Array) {
				n = len(current.Array)
			}
			return []Value{ArrayVal(current.Array[n:])}, nil
		}
		return []Value{current}, nil

	case "nth":
		if len(args) == 0 {
			return []Value{Null}, nil
		}
		nVals, err := e.evalExpr(args[0])
		if err != nil {
			return nil, err
		}
		n := 0
		if len(nVals) > 0 && nVals[0].Kind == ValNumber {
			n = int(nVals[0].Number)
		}
		if current.Kind == ValArray && n >= 0 && n < len(current.Array) {
			return []Value{current.Array[n]}, nil
		}
		return []Value{Null}, nil

	case "add":
		if current.Kind == ValArray {
			if len(current.Array) == 0 {
				return []Value{Null}, nil
			}
			acc := current.Array[0]
			for _, v := range current.Array[1:] {
				acc = addValues(acc, v)
			}
			return []Value{acc}, nil
		}
		return []Value{current}, nil

	case "min":
		if current.Kind == ValArray && len(current.Array) > 0 {
			m := current.Array[0]
			for _, v := range current.Array[1:] {
				if compareValues(v, m) < 0 {
					m = v
				}
			}
			return []Value{m}, nil
		}
		return []Value{Null}, nil

	case "max":
		if current.Kind == ValArray && len(current.Array) > 0 {
			m := current.Array[0]
			for _, v := range current.Array[1:] {
				if compareValues(v, m) > 0 {
					m = v
				}
			}
			return []Value{m}, nil
		}
		return []Value{Null}, nil

	case "group_by":
		if len(args) == 0 || current.Kind != ValArray {
			return []Value{current}, nil
		}
		keyArg := args[0]
		groups := make(map[string][]Value)
		var order []string
		for _, v := range current.Array {
			e.ctx.Current = v
			keyVals, err := e.evalExpr(keyArg)
			if err != nil {
				continue
			}
			key := ""
			if len(keyVals) > 0 {
				key = keyVals[0].ToText()
			}
			if _, exists := groups[key]; !exists {
				order = append(order, key)
			}
			groups[key] = append(groups[key], v)
		}
		e.ctx.Current = current
		result := make(map[string]Value)
		for _, k := range order {
			result[k] = ArrayVal(groups[k])
		}
		return []Value{{Kind: ValObject, Object: result}}, nil

	case "sort_by":
		if len(args) == 0 || current.Kind != ValArray {
			return []Value{current}, nil
		}
		keyArg := args[0]
		sorted := make([]Value, len(current.Array))
		copy(sorted, current.Array)
		sort.SliceStable(sorted, func(i, j int) bool {
			e.ctx.Current = sorted[i]
			ki, _ := e.evalExpr(keyArg)
			e.ctx.Current = sorted[j]
			kj, _ := e.evalExpr(keyArg)
			a := Null
			b := Null
			if len(ki) > 0 {
				a = ki[0]
			}
			if len(kj) > 0 {
				b = kj[0]
			}
			e.ctx.Current = current
			return compareValues(a, b) < 0
		})
		e.ctx.Current = current
		return []Value{ArrayVal(sorted)}, nil

	// --- String functions ---
	case "upper":
		return []Value{StringVal(strings.ToUpper(current.ToText()))}, nil
	case "lower":
		return []Value{StringVal(strings.ToLower(current.ToText()))}, nil
	case "trim":
		return []Value{StringVal(strings.TrimSpace(current.ToText()))}, nil
	case "slugify":
		slug := strings.ToLower(current.ToText())
		slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
		slug = strings.Trim(slug, "-")
		return []Value{StringVal(slug)}, nil
	case "split":
		if len(args) == 0 {
			return []Value{current}, nil
		}
		sepVals, err := e.evalExpr(args[0])
		if err != nil {
			return nil, err
		}
		sep := " "
		if len(sepVals) > 0 {
			sep = sepVals[0].ToText()
		}
		parts := strings.Split(current.ToText(), sep)
		arr := make([]Value, len(parts))
		for i, p := range parts {
			arr[i] = StringVal(p)
		}
		return []Value{ArrayVal(arr)}, nil

	case "join":
		sep := ""
		if len(args) > 0 {
			sepVals, err := e.evalExpr(args[0])
			if err != nil {
				return nil, err
			}
			if len(sepVals) > 0 {
				sep = sepVals[0].ToText()
			}
		}
		if current.Kind == ValArray {
			var parts []string
			for _, v := range current.Array {
				parts = append(parts, v.ToText())
			}
			return []Value{StringVal(strings.Join(parts, sep))}, nil
		}
		return []Value{current}, nil

	case "replace":
		if len(args) < 2 {
			return []Value{current}, nil
		}
		aVals, err := e.evalExpr(args[0])
		if err != nil {
			return nil, err
		}
		bVals, err := e.evalExpr(args[1])
		if err != nil {
			return nil, err
		}
		a := ""
		b := ""
		if len(aVals) > 0 {
			a = aVals[0].ToText()
		}
		if len(bVals) > 0 {
			b = bVals[0].ToText()
		}
		return []Value{StringVal(strings.ReplaceAll(current.ToText(), a, b))}, nil

	case "lines":
		lines := strings.Split(current.ToText(), "\n")
		return []Value{NumberVal(float64(len(lines)))}, nil
	case "words":
		return []Value{NumberVal(float64(len(strings.Fields(current.ToText()))))}, nil
	case "chars":
		return []Value{NumberVal(float64(len([]rune(current.ToText()))))}, nil

	// --- Filter functions ---
	case "select", "where", "filter":
		if len(args) == 0 {
			return []Value{current}, nil
		}
		condVals, err := e.evalExpr(args[0])
		if err != nil {
			return nil, err
		}
		cond := Null
		if len(condVals) > 0 {
			cond = condVals[0]
		}
		if cond.IsTruthy() {
			return []Value{current}, nil
		}
		return nil, nil

	case "contains", "includes":
		if len(args) == 0 {
			return []Value{BoolVal(false)}, nil
		}
		patVals, err := e.evalExpr(args[0])
		if err != nil {
			return nil, err
		}
		pat := ""
		if len(patVals) > 0 {
			pat = patVals[0].ToText()
		}
		return []Value{BoolVal(strings.Contains(current.ToText(), pat))}, nil

	case "startswith":
		if len(args) == 0 {
			return []Value{BoolVal(false)}, nil
		}
		patVals, err := e.evalExpr(args[0])
		if err != nil {
			return nil, err
		}
		pat := ""
		if len(patVals) > 0 {
			pat = patVals[0].ToText()
		}
		return []Value{BoolVal(strings.HasPrefix(current.ToText(), pat))}, nil

	case "endswith":
		if len(args) == 0 {
			return []Value{BoolVal(false)}, nil
		}
		patVals, err := e.evalExpr(args[0])
		if err != nil {
			return nil, err
		}
		pat := ""
		if len(patVals) > 0 {
			pat = patVals[0].ToText()
		}
		return []Value{BoolVal(strings.HasSuffix(current.ToText(), pat))}, nil

	case "matches":
		if len(args) == 0 {
			return []Value{BoolVal(false)}, nil
		}
		patVals, err := e.evalExpr(args[0])
		if err != nil {
			return nil, err
		}
		pat := ""
		if len(patVals) > 0 {
			pat = patVals[0].ToText()
		}
		re, err2 := regexp.Compile(pat)
		if err2 != nil {
			return nil, &EvalError{fmt.Sprintf("invalid regex: %v", err2)}
		}
		return []Value{BoolVal(re.MatchString(current.ToText()))}, nil

	case "any":
		if current.Kind == ValArray {
			for _, v := range current.Array {
				if v.IsTruthy() {
					return []Value{BoolVal(true)}, nil
				}
			}
			return []Value{BoolVal(false)}, nil
		}
		return []Value{BoolVal(current.IsTruthy())}, nil

	case "all":
		if current.Kind == ValArray {
			for _, v := range current.Array {
				if !v.IsTruthy() {
					return []Value{BoolVal(false)}, nil
				}
			}
			return []Value{BoolVal(true)}, nil
		}
		return []Value{BoolVal(current.IsTruthy())}, nil

	case "empty":
		if current.Kind == ValArray {
			return []Value{BoolVal(len(current.Array) == 0)}, nil
		}
		if current.Kind == ValString {
			return []Value{BoolVal(current.Str == "")}, nil
		}
		return []Value{BoolVal(current.Kind == ValNull)}, nil

	case "not":
		return []Value{BoolVal(!current.IsTruthy())}, nil

	// --- Aggregation ---
	case "stats":
		result := map[string]Value{
			"headings": NumberVal(float64(len(e.ctx.Headings))),
			"code":     NumberVal(float64(len(e.ctx.CodeBlocks))),
			"links":    NumberVal(float64(len(e.ctx.Links))),
			"images":   NumberVal(float64(len(e.ctx.Images))),
			"tables":   NumberVal(float64(len(e.ctx.Tables))),
			"words":    NumberVal(float64(e.ctx.Document.WordCount)),
		}
		return []Value{{Kind: ValObject, Object: result}}, nil

	case "levels":
		counts := make(map[string]Value)
		for _, h := range e.ctx.Headings {
			key := fmt.Sprintf("h%d", h.Level)
			if v, ok := counts[key]; ok {
				counts[key] = NumberVal(v.Number + 1)
			} else {
				counts[key] = NumberVal(1)
			}
		}
		return []Value{{Kind: ValObject, Object: counts}}, nil

	case "langs":
		counts := make(map[string]Value)
		for _, c := range e.ctx.CodeBlocks {
			lang := c.Language
			if lang == "" {
				lang = "unknown"
			}
			if v, ok := counts[lang]; ok {
				counts[lang] = NumberVal(v.Number + 1)
			} else {
				counts[lang] = NumberVal(1)
			}
		}
		return []Value{{Kind: ValObject, Object: counts}}, nil

	case "types":
		counts := make(map[string]Value)
		for _, l := range e.ctx.Links {
			if v, ok := counts[l.LinkType]; ok {
				counts[l.LinkType] = NumberVal(v.Number + 1)
			} else {
				counts[l.LinkType] = NumberVal(1)
			}
		}
		return []Value{{Kind: ValObject, Object: counts}}, nil

	default:
		return nil, &EvalError{fmt.Sprintf("unknown function %q", name)}
	}
}
