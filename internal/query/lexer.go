package query

import (
	"fmt"
	"strings"
)

// TokenKind identifies the type of a lexer token.
type TokenKind int

const (
	TokDot TokenKind = iota
	TokPipe
	TokComma
	TokColon
	TokLBracket
	TokRBracket
	TokLParen
	TokRParen
	TokLBrace
	TokRBrace
	TokGt
	TokGtGt
	TokQuestion

	TokEq
	TokNe
	TokLt
	TokLe
	TokGe
	TokPlus
	TokMinus
	TokStar
	TokSlash
	TokPercent
	TokSlashSlash

	TokKwAnd
	TokKwOr
	TokKwNot
	TokKwIf
	TokKwThen
	TokKwElif
	TokKwElse
	TokKwEnd
	TokKwTrue
	TokKwFalse
	TokKwNull

	TokString
	TokNumber
	TokRegex
	TokIdent

	TokEOF
)

func (k TokenKind) String() string {
	switch k {
	case TokDot:
		return "'.'"
	case TokPipe:
		return "'|'"
	case TokComma:
		return "','"
	case TokColon:
		return "':'"
	case TokLBracket:
		return "'['"
	case TokRBracket:
		return "']'"
	case TokLParen:
		return "'('"
	case TokRParen:
		return "')'"
	case TokLBrace:
		return "'{'"
	case TokRBrace:
		return "'}'"
	case TokGt:
		return "'>'"
	case TokGtGt:
		return "'>>'"
	case TokQuestion:
		return "'?'"
	case TokEq:
		return "'=='"
	case TokNe:
		return "'!='"
	case TokLt:
		return "'<'"
	case TokLe:
		return "'<='"
	case TokGe:
		return "'>='"
	case TokPlus:
		return "'+'"
	case TokMinus:
		return "'-'"
	case TokStar:
		return "'*'"
	case TokSlash:
		return "'/'"
	case TokPercent:
		return "'%'"
	case TokSlashSlash:
		return "'//'"
	case TokKwAnd:
		return "'and'"
	case TokKwOr:
		return "'or'"
	case TokKwNot:
		return "'not'"
	case TokKwIf:
		return "'if'"
	case TokKwThen:
		return "'then'"
	case TokKwElif:
		return "'elif'"
	case TokKwElse:
		return "'else'"
	case TokKwEnd:
		return "'end'"
	case TokKwTrue:
		return "'true'"
	case TokKwFalse:
		return "'false'"
	case TokKwNull:
		return "'null'"
	case TokString:
		return "string"
	case TokNumber:
		return "number"
	case TokRegex:
		return "regex"
	case TokIdent:
		return "identifier"
	case TokEOF:
		return "end of input"
	}
	return "unknown"
}

// Token is a lexed token with its source span.
type Token struct {
	Kind   TokenKind
	Span   Span
	StrVal string  // for TokString, TokIdent, TokRegex
	NumVal float64 // for TokNumber
}

// LexError is a lexer error.
type LexError struct {
	Message string
	Span    Span
	Input   string
}

func (e *LexError) Error() string {
	if e.Span.Start < len(e.Input) {
		return fmt.Sprintf("lex error at position %d: %s", e.Span.Start, e.Message)
	}
	return fmt.Sprintf("lex error: %s", e.Message)
}

type lexer struct {
	input []rune
	pos   int
}

func newLexer(input string) *lexer {
	return &lexer{input: []rune(input)}
}

func (l *lexer) peek() (rune, bool) {
	if l.pos >= len(l.input) {
		return 0, false
	}
	return l.input[l.pos], true
}

func (l *lexer) advance() (rune, bool) {
	if l.pos >= len(l.input) {
		return 0, false
	}
	c := l.input[l.pos]
	l.pos++
	return c, true
}

func (l *lexer) skipWhitespace() {
	for {
		c, ok := l.peek()
		if !ok || !isWhitespace(c) {
			break
		}
		l.advance()
	}
}

func isWhitespace(c rune) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func (l *lexer) readString(quote rune, start int) (Token, error) {
	var sb strings.Builder
	for {
		c, ok := l.advance()
		if !ok {
			return Token{}, &LexError{"unterminated string", newSpan(start, l.pos), ""}
		}
		if c == quote {
			return Token{Kind: TokString, Span: newSpan(start, l.pos), StrVal: sb.String()}, nil
		}
		if c == '\\' {
			esc, ok := l.advance()
			if !ok {
				return Token{}, &LexError{"unterminated escape", newSpan(start, l.pos), ""}
			}
			switch esc {
			case 'n':
				sb.WriteRune('\n')
			case 'r':
				sb.WriteRune('\r')
			case 't':
				sb.WriteRune('\t')
			case '\\':
				sb.WriteRune('\\')
			default:
				if esc == quote {
					sb.WriteRune(quote)
				} else {
					return Token{}, &LexError{fmt.Sprintf("invalid escape \\%c", esc), newSpan(l.pos-1, l.pos), ""}
				}
			}
		} else {
			sb.WriteRune(c)
		}
	}
}

func (l *lexer) readNumber(start int, first rune) Token {
	var sb strings.Builder
	sb.WriteRune(first)
	for {
		c, ok := l.peek()
		if !ok {
			break
		}
		if c >= '0' && c <= '9' || c == '.' || c == 'e' || c == 'E' {
			sb.WriteRune(c)
			l.advance()
		} else if (c == '-' || c == '+') && (sb.String()[len(sb.String())-1] == 'e' || sb.String()[len(sb.String())-1] == 'E') {
			sb.WriteRune(c)
			l.advance()
		} else {
			break
		}
	}
	var val float64
	fmt.Sscanf(sb.String(), "%f", &val)
	return Token{Kind: TokNumber, Span: newSpan(start, l.pos), NumVal: val}
}

func (l *lexer) readIdent(start int, first rune) Token {
	var sb strings.Builder
	sb.WriteRune(first)
	for {
		c, ok := l.peek()
		if !ok {
			break
		}
		if isIdentChar(c) {
			sb.WriteRune(c)
			l.advance()
		} else {
			break
		}
	}
	ident := sb.String()
	kind := TokIdent
	switch ident {
	case "and":
		kind = TokKwAnd
	case "or":
		kind = TokKwOr
	case "not":
		kind = TokKwNot
	case "if":
		kind = TokKwIf
	case "then":
		kind = TokKwThen
	case "elif":
		kind = TokKwElif
	case "else":
		kind = TokKwElse
	case "end":
		kind = TokKwEnd
	case "true":
		kind = TokKwTrue
	case "false":
		kind = TokKwFalse
	case "null":
		kind = TokKwNull
	}
	return Token{Kind: kind, Span: newSpan(start, l.pos), StrVal: ident}
}

func isIdentChar(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

func (l *lexer) nextToken() (Token, error) {
	l.skipWhitespace()
	start := l.pos

	c, ok := l.advance()
	if !ok {
		return Token{Kind: TokEOF, Span: newSpan(l.pos, l.pos)}, nil
	}

	switch c {
	case '.':
		return Token{Kind: TokDot, Span: newSpan(start, l.pos)}, nil
	case '|':
		return Token{Kind: TokPipe, Span: newSpan(start, l.pos)}, nil
	case ',':
		return Token{Kind: TokComma, Span: newSpan(start, l.pos)}, nil
	case ':':
		return Token{Kind: TokColon, Span: newSpan(start, l.pos)}, nil
	case '[':
		return Token{Kind: TokLBracket, Span: newSpan(start, l.pos)}, nil
	case ']':
		return Token{Kind: TokRBracket, Span: newSpan(start, l.pos)}, nil
	case '(':
		return Token{Kind: TokLParen, Span: newSpan(start, l.pos)}, nil
	case ')':
		return Token{Kind: TokRParen, Span: newSpan(start, l.pos)}, nil
	case '{':
		return Token{Kind: TokLBrace, Span: newSpan(start, l.pos)}, nil
	case '}':
		return Token{Kind: TokRBrace, Span: newSpan(start, l.pos)}, nil
	case '?':
		return Token{Kind: TokQuestion, Span: newSpan(start, l.pos)}, nil
	case '+':
		return Token{Kind: TokPlus, Span: newSpan(start, l.pos)}, nil
	case '*':
		return Token{Kind: TokStar, Span: newSpan(start, l.pos)}, nil
	case '%':
		return Token{Kind: TokPercent, Span: newSpan(start, l.pos)}, nil
	case '-':
		if nc, ok2 := l.peek(); ok2 && nc >= '0' && nc <= '9' {
			return l.readNumber(start, c), nil
		}
		return Token{Kind: TokMinus, Span: newSpan(start, l.pos)}, nil
	case '>':
		if nc, ok2 := l.peek(); ok2 && nc == '>' {
			l.advance()
			return Token{Kind: TokGtGt, Span: newSpan(start, l.pos)}, nil
		} else if ok2 && nc == '=' {
			l.advance()
			return Token{Kind: TokGe, Span: newSpan(start, l.pos)}, nil
		}
		return Token{Kind: TokGt, Span: newSpan(start, l.pos)}, nil
	case '<':
		if nc, ok2 := l.peek(); ok2 && nc == '=' {
			l.advance()
			return Token{Kind: TokLe, Span: newSpan(start, l.pos)}, nil
		}
		return Token{Kind: TokLt, Span: newSpan(start, l.pos)}, nil
	case '=':
		if nc, ok2 := l.peek(); ok2 && nc == '=' {
			l.advance()
			return Token{Kind: TokEq, Span: newSpan(start, l.pos)}, nil
		}
		return Token{}, &LexError{"use '==' for equality", newSpan(start, l.pos), ""}
	case '!':
		if nc, ok2 := l.peek(); ok2 && nc == '=' {
			l.advance()
			return Token{Kind: TokNe, Span: newSpan(start, l.pos)}, nil
		}
		return Token{}, &LexError{"use 'not' for negation or '!=' for inequality", newSpan(start, l.pos), ""}
	case '/':
		if nc, ok2 := l.peek(); ok2 && nc == '/' {
			l.advance()
			return Token{Kind: TokSlashSlash, Span: newSpan(start, l.pos)}, nil
		}
		return Token{Kind: TokSlash, Span: newSpan(start, l.pos)}, nil
	case '"', '\'':
		return l.readString(c, start)
	default:
		if c >= '0' && c <= '9' {
			return l.readNumber(start, c), nil
		}
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' {
			return l.readIdent(start, c), nil
		}
		return Token{}, &LexError{fmt.Sprintf("unexpected character %q", c), newSpan(start, l.pos), ""}
	}
}

// Tokenize converts an input string to a token slice.
func Tokenize(input string) ([]Token, error) {
	l := newLexer(input)
	var tokens []Token
	for {
		tok, err := l.nextToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.Kind == TokEOF {
			break
		}
	}
	return tokens, nil
}
