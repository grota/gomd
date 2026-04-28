package query

import (
	"fmt"
	"strings"
)

// ParseError is a parser error.
type ParseError struct {
	Message string
	Span    Span
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error at position %d: %s", e.Span.Start, e.Message)
}

type parser struct {
	tokens []Token
	pos    int
}

func newParser(tokens []Token) *parser {
	return &parser{tokens: tokens}
}

func (p *parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Kind: TokEOF}
	}
	return p.tokens[p.pos]
}

func (p *parser) advance() Token {
	tok := p.peek()
	if tok.Kind != TokEOF {
		p.pos++
	}
	return tok
}

func (p *parser) expect(kind TokenKind) (Token, error) {
	tok := p.peek()
	if tok.Kind != kind {
		return tok, &ParseError{
			fmt.Sprintf("expected %s, got %s", kind, tok.Kind),
			tok.Span,
		}
	}
	return p.advance(), nil
}

func (p *parser) check(kind TokenKind) bool {
	return p.peek().Kind == kind
}

func (p *parser) match(kinds ...TokenKind) bool {
	for _, k := range kinds {
		if p.peek().Kind == k {
			return true
		}
	}
	return false
}

// Parse parses a query string.
func Parse(input string) (*Query, error) {
	tokens, err := Tokenize(input)
	if err != nil {
		return nil, err
	}
	p := newParser(tokens)
	return p.parseQuery()
}

func (p *parser) parseQuery() (*Query, error) {
	var exprs []PipedExpr

	piped, err := p.parsePipedExpr()
	if err != nil {
		return nil, err
	}
	exprs = append(exprs, piped)

	for p.check(TokComma) {
		p.advance()
		piped, err = p.parsePipedExpr()
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, piped)
	}

	if !p.check(TokEOF) {
		tok := p.peek()
		return nil, &ParseError{fmt.Sprintf("unexpected token %s", tok.Kind), tok.Span}
	}

	return &Query{Expressions: exprs}, nil
}

func (p *parser) parsePipedExpr() (PipedExpr, error) {
	var stages []Expr

	stage, err := p.parseExprBinary(0)
	if err != nil {
		return PipedExpr{}, err
	}
	stages = append(stages, stage)

	for p.check(TokPipe) {
		p.advance()
		stage, err = p.parseExprBinary(0)
		if err != nil {
			return PipedExpr{}, err
		}
		stages = append(stages, stage)
	}

	return PipedExpr{Stages: stages}, nil
}

func (p *parser) parseExprBinary(minPrec int) (Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for {
		op, ok := p.peekBinaryOp()
		if !ok || op.Precedence() <= minPrec {
			break
		}

		// Check for hierarchy operators first
		if p.check(TokGt) || p.check(TokGtGt) {
			direct := p.check(TokGt)
			opTok := p.advance()
			right, err := p.parseUnary()
			if err != nil {
				return nil, err
			}
			left = &HierarchyExpr{
				baseExpr: baseExpr{left.exprSpan().merge(right.exprSpan())},
				Parent:   left,
				Child:    right,
				Direct:   direct,
			}
			_ = opTok
			continue
		}

		p.advance() // consume operator
		right, err := p.parseExprBinary(op.Precedence())
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{
			baseExpr: baseExpr{left.exprSpan().merge(right.exprSpan())},
			Op:       op,
			Left:     left,
			Right:    right,
		}
	}

	return left, nil
}

func (p *parser) peekBinaryOp() (BinaryOp, bool) {
	switch p.peek().Kind {
	case TokEq:
		return BinEq, true
	case TokNe:
		return BinNe, true
	case TokLt:
		return BinLt, true
	case TokLe:
		return BinLe, true
	case TokGt:
		return BinGt, true
	case TokGe:
		return BinGe, true
	case TokKwAnd:
		return BinAnd, true
	case TokKwOr:
		return BinOr, true
	case TokPlus:
		return BinAdd, true
	case TokMinus:
		return BinSub, true
	case TokStar:
		return BinMul, true
	case TokSlash:
		return BinDiv, true
	case TokPercent:
		return BinMod, true
	case TokSlashSlash:
		return BinAlt, true
	}
	return 0, false
}

func (p *parser) parseUnary() (Expr, error) {
	tok := p.peek()
	if tok.Kind == TokKwNot {
		p.advance()
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{baseExpr: baseExpr{tok.Span.merge(expr.exprSpan())}, Op: UnaryNot, Expr: expr}, nil
	}
	if tok.Kind == TokMinus {
		p.advance()
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{baseExpr: baseExpr{tok.Span.merge(expr.exprSpan())}, Op: UnaryNeg, Expr: expr}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (Expr, error) {
	tok := p.peek()

	switch tok.Kind {
	case TokDot:
		return p.parseDotExpr()

	case TokLParen:
		p.advance()
		inner, err := p.parseExprBinary(0)
		if err != nil {
			return nil, err
		}
		_, err = p.expect(TokRParen)
		if err != nil {
			return nil, err
		}
		return &GroupExpr{baseExpr: baseExpr{tok.Span.merge(inner.exprSpan())}, Expr: inner}, nil

	case TokLBracket:
		return p.parseArrayExpr()

	case TokLBrace:
		return p.parseObjectExpr()

	case TokKwIf:
		return p.parseConditional()

	case TokString:
		p.advance()
		return &LiteralExpr{baseExpr{tok.Span}, LiteralValue{Kind: LitString, Str: tok.StrVal}}, nil

	case TokNumber:
		p.advance()
		return &LiteralExpr{baseExpr{tok.Span}, LiteralValue{Kind: LitNumber, Num: tok.NumVal}}, nil

	case TokKwTrue:
		p.advance()
		return &LiteralExpr{baseExpr{tok.Span}, LiteralValue{Kind: LitBool, Bool: true}}, nil

	case TokKwFalse:
		p.advance()
		return &LiteralExpr{baseExpr{tok.Span}, LiteralValue{Kind: LitBool, Bool: false}}, nil

	case TokKwNull:
		p.advance()
		return &LiteralExpr{baseExpr{tok.Span}, LiteralValue{Kind: LitNull}}, nil

	case TokIdent:
		return p.parseFunctionOrIdent()

	default:
		return nil, &ParseError{fmt.Sprintf("unexpected token %s", tok.Kind), tok.Span}
	}
}

func (p *parser) parseDotExpr() (Expr, error) {
	dotTok := p.advance() // consume '.'

	// Check what follows
	next := p.peek()
	if next.Kind == TokIdent {
		ident := p.advance()
		name := ident.StrVal

		// Check if it's an element kind
		if spec, ok := ParseElementKind(name); ok {
			elem := &ElementExpr{
				baseExpr: baseExpr{dotTok.Span.merge(ident.Span)},
				Kind:     spec.Kind,
			}
			if spec.Kind == ElemHeading {
				elem.Kind = ElemHeading
				// Store heading level in a special way via filters if needed
				// We'll use the ElementSpec to embed it
				_ = spec.HeadingLevel
				// Actually we need to store this; let's use a custom approach
				// Store heading level as part of the element expr
				return p.finishElementExpr(elem, spec.HeadingLevel)
			}
			return p.finishElementExpr(elem, 0)
		}

		// Otherwise it's a property access
		return &PropertyExpr{baseExpr{dotTok.Span.merge(ident.Span)}, name}, nil
	}

	// Just "." - identity
	return &IdentityExpr{baseExpr{dotTok.Span}}, nil
}

// headingLevelElement wraps ElementExpr with a heading level.
// We'll embed the level inside the ElementExpr using a custom field approach.
// Since Go doesn't allow embedding differently, we use a wrapper.
type headingElementExpr struct {
	*ElementExpr
	HeadingLevel int
}

func (e *headingElementExpr) exprNode() {}
func (e *headingElementExpr) exprSpan() Span { return e.ElementExpr.span }

func (p *parser) finishElementExpr(elem *ElementExpr, headingLevel int) (Expr, error) {
	var filters []Filter
	var indexOp *IndexOp

	// Parse optional bracket filters/indices
	for p.check(TokLBracket) {
		p.advance()
		next := p.peek()
		switch next.Kind {
		case TokRBracket:
			// [] - iterate
			p.advance()
			op := &IndexOp{Kind: IndexIterate}
			indexOp = op
		case TokNumber:
			// [0] or [0:3] - index or slice
			numTok := p.advance()
			idx := int64(numTok.NumVal)
			if p.check(TokColon) {
				p.advance()
				if p.check(TokRBracket) {
					p.advance()
					indexOp = &IndexOp{Kind: IndexSlice, Start: &idx}
				} else if p.check(TokNumber) {
					endTok := p.advance()
					endIdx := int64(endTok.NumVal)
					_, err := p.expect(TokRBracket)
					if err != nil {
						return nil, err
					}
					indexOp = &IndexOp{Kind: IndexSlice, Start: &idx, End: &endIdx}
				}
			} else {
				_, err := p.expect(TokRBracket)
				if err != nil {
					return nil, err
				}
				indexOp = &IndexOp{Kind: IndexSingle, Index: idx}
			}
		case TokColon:
			// [:3] - slice from start
			p.advance()
			if p.check(TokNumber) {
				endTok := p.advance()
				endIdx := int64(endTok.NumVal)
				_, err := p.expect(TokRBracket)
				if err != nil {
					return nil, err
				}
				indexOp = &IndexOp{Kind: IndexSlice, End: &endIdx}
			}
		case TokString:
			// ["exact text"] - exact text filter
			strTok := p.advance()
			_, err := p.expect(TokRBracket)
			if err != nil {
				return nil, err
			}
			filters = append(filters, &TextFilter{Pattern: strTok.StrVal, Exact: true, Span: strTok.Span})
		case TokSlash:
			// [/regex/] - regex filter
			p.advance()
			var sb strings.Builder
			for !p.check(TokSlash) && !p.check(TokEOF) {
				tok := p.advance()
				sb.WriteString(tok.StrVal)
			}
			_, err := p.expect(TokSlash)
			if err != nil {
				return nil, err
			}
			_, err = p.expect(TokRBracket)
			if err != nil {
				return nil, err
			}
			filters = append(filters, &RegexFilter{Pattern: sb.String(), Span: next.Span})
		default:
			// [identifier] or sequence - text filter
			var parts []string
			for !p.check(TokRBracket) && !p.check(TokEOF) {
				tok := p.advance()
				if tok.Kind == TokIdent {
					parts = append(parts, tok.StrVal)
				} else if tok.Kind == TokKwAnd || tok.Kind == TokKwOr || tok.Kind == TokKwNot ||
					tok.Kind == TokKwTrue || tok.Kind == TokKwFalse || tok.Kind == TokKwNull {
					parts = append(parts, tok.StrVal)
				} else {
					parts = append(parts, tok.StrVal)
				}
			}
			_, err := p.expect(TokRBracket)
			if err != nil {
				return nil, err
			}
			pattern := strings.Join(parts, " ")
			filters = append(filters, &TextFilter{Pattern: pattern, Exact: false, Span: next.Span})
		}
	}

	elem.Filters = filters
	elem.Index = indexOp

	if headingLevel != 0 {
		return &headingElementExpr{ElementExpr: elem, HeadingLevel: headingLevel}, nil
	}
	return elem, nil
}

func (p *parser) parseFunctionOrIdent() (Expr, error) {
	tok := p.advance()
	name := tok.StrVal

	// If followed by '(', it's a function call
	if p.check(TokLParen) {
		p.advance()
		var args []Expr
		if !p.check(TokRParen) {
			arg, err := p.parseExprBinary(0)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
			for p.check(TokComma) {
				p.advance()
				arg, err = p.parseExprBinary(0)
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
			}
		}
		closeTok, err := p.expect(TokRParen)
		if err != nil {
			return nil, err
		}
		return &FunctionExpr{
			baseExpr: baseExpr{tok.Span.merge(closeTok.Span)},
			Name:     name,
			Args:     args,
		}, nil
	}

	// Otherwise it's a bare function call with no args (like `count`, `text`, `upper`)
	return &FunctionExpr{
		baseExpr: baseExpr{tok.Span},
		Name:     name,
	}, nil
}

func (p *parser) parseArrayExpr() (Expr, error) {
	openTok := p.advance() // '['
	var elements []Expr
	if !p.check(TokRBracket) {
		elem, err := p.parsePipedExprAsExpr()
		if err != nil {
			return nil, err
		}
		elements = append(elements, elem)
		for p.check(TokComma) {
			p.advance()
			elem, err = p.parsePipedExprAsExpr()
			if err != nil {
				return nil, err
			}
			elements = append(elements, elem)
		}
	}
	closeTok, err := p.expect(TokRBracket)
	if err != nil {
		return nil, err
	}
	return &ArrayExpr{baseExpr{openTok.Span.merge(closeTok.Span)}, elements}, nil
}

func (p *parser) parsePipedExprAsExpr() (Expr, error) {
	piped, err := p.parsePipedExpr()
	if err != nil {
		return nil, err
	}
	if len(piped.Stages) == 1 {
		return piped.Stages[0], nil
	}
	// Wrap in a function
	return &FunctionExpr{
		baseExpr: baseExpr{piped.Stages[0].exprSpan().merge(piped.Stages[len(piped.Stages)-1].exprSpan())},
		Name:     "_pipe",
		Args:     piped.Stages,
	}, nil
}

func (p *parser) parseObjectExpr() (Expr, error) {
	openTok := p.advance() // '{'
	var pairs []ObjectPair
	if !p.check(TokRBrace) {
		pair, err := p.parseObjectPair()
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, pair)
		for p.check(TokComma) {
			p.advance()
			if p.check(TokRBrace) {
				break
			}
			pair, err = p.parseObjectPair()
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, pair)
		}
	}
	closeTok, err := p.expect(TokRBrace)
	if err != nil {
		return nil, err
	}
	return &ObjectExpr{baseExpr{openTok.Span.merge(closeTok.Span)}, pairs}, nil
}

func (p *parser) parseObjectPair() (ObjectPair, error) {
	// key: expr
	keyTok := p.peek()
	var key string
	if keyTok.Kind == TokIdent || keyTok.Kind == TokString {
		p.advance()
		key = keyTok.StrVal
	} else {
		return ObjectPair{}, &ParseError{fmt.Sprintf("expected object key, got %s", keyTok.Kind), keyTok.Span}
	}
	_, err := p.expect(TokColon)
	if err != nil {
		return ObjectPair{}, err
	}
	val, err := p.parseExprBinary(0)
	if err != nil {
		return ObjectPair{}, err
	}
	return ObjectPair{Key: key, Value: val}, nil
}

func (p *parser) parseConditional() (Expr, error) {
	ifTok := p.advance() // 'if'
	cond, err := p.parseExprBinary(0)
	if err != nil {
		return nil, err
	}
	_, err = p.expect(TokKwThen)
	if err != nil {
		return nil, err
	}
	thenBranch, err := p.parseExprBinary(0)
	if err != nil {
		return nil, err
	}
	var elseBranch Expr
	if p.check(TokKwElse) {
		p.advance()
		elseBranch, err = p.parseExprBinary(0)
		if err != nil {
			return nil, err
		}
	} else if p.check(TokKwElif) {
		elseBranch, err = p.parseConditional()
		if err != nil {
			return nil, err
		}
	}
	endTok, err := p.expect(TokKwEnd)
	if err != nil {
		return nil, err
	}
	return &ConditionalExpr{
		baseExpr:   baseExpr{ifTok.Span.merge(endTok.Span)},
		Condition:  cond,
		ThenBranch: thenBranch,
		ElseBranch: elseBranch,
	}, nil
}
