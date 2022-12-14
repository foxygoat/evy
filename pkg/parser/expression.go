// This file contains a Top Down Operator Precedence or Pratt parser.
//
// It is based on Thorston Ball's monkey interpreter:
// https://github.com/juliaogris/monkey/blob/master/parser/parser.go
//
// The expression parser is implemented in receiver functions of the
// Parser struct as defined in parser.go.

package parser

import (
	"strconv"

	"foxygo.at/evy/pkg/lexer"
)

type precedence int

const (
	LOWEST      precedence = iota
	OR                     // or
	AND                    // and
	EQUALS                 // ==
	LESSGREATER            // > or <
	SUM                    // +
	PRODUCT                // *
	UNARY                  // -x  !x
	INDEX                  // array[i]
)

var precedences = map[lexer.TokenType]precedence{
	lexer.EQ:       EQUALS,
	lexer.NOT_EQ:   EQUALS,
	lexer.LT:       LESSGREATER,
	lexer.GT:       LESSGREATER,
	lexer.LTEQ:     LESSGREATER,
	lexer.GTEQ:     LESSGREATER,
	lexer.PLUS:     SUM,
	lexer.MINUS:    SUM,
	lexer.OR:       OR,
	lexer.SLASH:    PRODUCT,
	lexer.ASTERISK: PRODUCT,
	lexer.AND:      AND,
	lexer.LBRACKET: INDEX,
	lexer.DOT:      INDEX,
}

func (p *Parser) parseTopLevelExpr(scope *scope) Node {
	tok := p.cur
	if tok.Type == lexer.IDENT && p.funcs[tok.Literal] != nil {
		return p.parseFuncCall(scope)
	}
	return p.parseExpr(scope, LOWEST)
}

func (p *Parser) parseFuncCall(scope *scope) Node {
	fc := &FunctionCall{Token: p.cur, Name: p.cur.Literal}
	p.advance() // advance past function name IDENT
	fc.FuncDecl = p.funcs[fc.Name]
	fc.Arguments = p.parseExprList(scope)
	p.assertArgTypes(fc.FuncDecl, fc.Arguments)
	return fc
}

func (p *Parser) parseExpr(scope *scope, prec precedence) Node {
	var left Node
	switch p.cur.Type {
	case lexer.IDENT:
		left = p.lookupVar(scope)
	case lexer.STRING_LIT, lexer.NUM_LIT, lexer.TRUE, lexer.FALSE, lexer.LBRACKET, lexer.LCURLY:
		left = p.parseLiteral(scope)
	case lexer.BANG, lexer.MINUS:
		left = p.parseUnaryExpr(scope)
	case lexer.LPAREN:
		left = p.parseGroupedExpr(scope)
	default:
		p.unexpectedLeftTokenError()
	}
	for left != nil && !p.isAtExprEnd() && prec < precedences[p.cur.Type] {
		tt := p.cur.Type
		switch {
		case isBinaryOp(tt):
			left = p.parseBinaryExpr(scope, left)
		case tt == lexer.LBRACKET:
			left = p.parseIndexOrSliceExpr(scope, left, true)
		case tt == lexer.DOT:
			left = p.parseDotExpr(left)
		default:
			return left
		}
	}
	return left // nil for previous error
}

func (p *Parser) unexpectedLeftTokenError() {
	if p.isWSS() {
		tt := p.cur.Type
		prevTT := p.lookAt(p.pos - 1).Type
		if isBinaryOp(tt) && prevTT == lexer.WS {
			p.appendError("unexpected whitespace before " + p.cur.FormatDetails())
			return
		}
		if tt == lexer.WS && isBinaryOp(prevTT) {
			prevToken := p.lookAt(p.pos - 1)
			p.appendErrorForToken("unexpected whitespace after "+prevToken.FormatDetails(), prevToken)
			return
		}
	}
	p.appendError("unexpected " + p.cur.FormatDetails())
}

func (p *Parser) isAtExprEnd() bool {
	if p.isWSS() && p.cur.Type == lexer.WS {
		return true
	}
	return p.isAtEOL()
}

func (p *Parser) parseUnaryExpr(scope *scope) Node {
	tok := p.cur
	unaryExp := &UnaryExpression{Token: tok, Op: op(tok)}
	p.advance() // advance past operator
	if p.lookAt(p.pos-1).Type == lexer.WS {
		p.appendErrorForToken("unexpected whitespace after '"+unaryExp.Op.String()+"'", tok)
	}
	unaryExp.Right = p.parseExpr(scope, UNARY)
	if unaryExp.Right == nil {
		return nil // previous error
	}
	p.validateUnaryType(unaryExp)
	return unaryExp
}

func (p *Parser) parseBinaryExpr(scope *scope, left Node) Node {
	tok := p.cur
	expType := left.Type()
	if isComparisonOp(tok.Type) {
		expType = BOOL_TYPE
	}
	binaryExp := &BinaryExpression{Token: tok, T: expType, Op: op(tok), Left: left}
	prec := precedences[tok.Type]
	p.advance() // advance past operator
	binaryExp.Right = p.parseExpr(scope, prec)
	if binaryExp.Right == nil {
		return nil // previous error
	}
	p.validateBinaryType(binaryExp)
	return binaryExp
}

func (p *Parser) parseGroupedExpr(scope *scope) Node {
	p.pushWSS(false)
	defer p.popWSS()
	p.advance() // advance past (
	exp := p.parseTopLevelExpr(scope)
	if !p.assertToken(lexer.RPAREN) {
		return nil
	}
	p.advance() // advance past )
	return exp
}

func (p *Parser) parseIndexOrSliceExpr(scope *scope, left Node, allowSlice bool) Node {
	p.pushWSS(false)
	defer p.popWSS()
	tok := p.cur
	if p.lookAt(p.pos-1).Type == lexer.WS {
		p.appendError("unexpected whitespace before '['")
		return nil
	}
	p.advance() // advance past [
	leftType := left.Type().Name
	if leftType != ARRAY && leftType != MAP && leftType != STRING {
		p.appendErrorForToken("only array, string and map type can be indexed found "+left.Type().Format(), tok)
		return nil
	}
	if p.cur.TokenType() == lexer.COLON && allowSlice { // e.g. a[:2]
		p.advance() //  advance past :
		return p.parseSlice(scope, tok, left, nil)
	}
	index := p.parseTopLevelExpr(scope)
	if index == nil {
		return nil
	}
	tt := p.cur.TokenType()
	if tt == lexer.COLON && allowSlice { // e.g. a[1:3] or a[1:]
		p.advance() // advance past :
		return p.parseSlice(scope, tok, left, index)
	}
	if !p.validateIndex(tok, leftType, index.Type()) {
		return nil
	}
	p.advance() // advance past ]
	t := left.Type().Sub
	if leftType == STRING {
		t = STRING_TYPE
	}
	return &IndexExpression{Token: tok, Left: left, Index: index, T: t}
}

func (p *Parser) validateIndex(tok *lexer.Token, leftType TypeName, indexType *Type) bool {
	if !p.assertToken(lexer.RBRACKET) {
		return false
	}
	if (leftType == ARRAY || leftType == STRING) && indexType != NUM_TYPE {
		p.appendErrorForToken(leftType.String()+" index expects num, found "+indexType.Format(), tok)
		return false
	}
	if leftType == MAP && indexType != STRING_TYPE {
		p.appendErrorForToken("map index expects string, found "+indexType.Format(), tok)
		return false
	}
	return true
}

func (p *Parser) parseSlice(scope *scope, tok *lexer.Token, left, start Node) Node {
	leftType := left.Type().Name
	if leftType != ARRAY && leftType != STRING {
		p.appendErrorForToken("only array and string be indexed sliced"+left.Type().Format(), tok)
		return nil
	}

	t := left.Type()
	if p.cur.Type == lexer.RBRACKET {
		p.advance()
		return &SliceExpression{Token: tok, Left: left, Start: start, End: nil, T: t}
	}
	end := p.parseTopLevelExpr(scope)
	if end == nil {
		return nil
	}
	if !p.assertToken(lexer.RBRACKET) {
		return nil
	}
	p.advance()
	return &SliceExpression{Token: tok, Left: left, Start: start, End: end, T: t}
}

func (p *Parser) parseDotExpr(left Node) Node {
	tok := p.cur
	if p.lookAt(p.pos-1).Type == lexer.WS {
		p.appendError("unexpected whitespace before '.'")
		return nil
	}
	if p.lookAt(p.pos+1).Type == lexer.WS {
		p.appendError("unexpected whitespace after '.'")
		return nil
	}
	p.advance() // advance past .
	leftType := left.Type().Name
	if leftType != MAP {
		p.appendErrorForToken("field access with '.' expects map type, found "+left.Type().Format(), tok)
		return nil
	}
	if p.cur.TokenType() != lexer.IDENT {
		p.appendErrorForToken("expected map key, found "+p.cur.TokenType().Format(), tok)
		return nil
	}
	expr := &DotExpression{Token: tok, Left: left, T: left.Type().Sub, Key: p.cur.Literal}
	p.advance() // advance past key IDENT
	return expr
}

func isBinaryOp(tt lexer.TokenType) bool {
	return isComparisonOp(tt) || tt == lexer.PLUS || tt == lexer.MINUS || tt == lexer.SLASH || tt == lexer.ASTERISK || tt == lexer.OR || tt == lexer.AND
}

func isComparisonOp(tt lexer.TokenType) bool {
	return tt == lexer.EQ || tt == lexer.NOT_EQ || tt == lexer.LT || tt == lexer.GT || tt == lexer.LTEQ || tt == lexer.GTEQ
}

func (p *Parser) validateUnaryType(unaryExp *UnaryExpression) {
	tok := unaryExp.Token
	rightType := unaryExp.Right.Type()
	switch unaryExp.Op {
	case OP_MINUS:
		if unaryExp.Right.Type() != NUM_TYPE {
			p.appendErrorForToken("'-' unary expects num type, found "+rightType.String(), tok)
		}
	case OP_BANG:
		if unaryExp.Right.Type() != BOOL_TYPE {
			p.appendErrorForToken("'!' unary expects bool type, found "+rightType.String(), tok)
		}
	default:
		p.appendErrorForToken("invalid unary operator", tok)
	}
}

func (p *Parser) validateBinaryType(binaryExp *BinaryExpression) {
	tok := binaryExp.Token
	op := binaryExp.Op
	if op == OP_ILLEGAL || op == OP_BANG {
		p.appendErrorForToken("invalid binary operator", tok)
		return
	}

	leftType := binaryExp.Left.Type()
	rightType := binaryExp.Right.Type()
	if !leftType.Matches(rightType) {
		p.appendErrorForToken("mismatched type for "+op.String()+": "+leftType.Format()+", "+rightType.Format(), tok)
		return
	}

	switch op {
	case OP_PLUS:
		if leftType != NUM_TYPE && leftType != STRING_TYPE && leftType.Name != ARRAY {
			p.appendErrorForToken("'+' takes num, string or array type, found "+leftType.Format(), tok)
		}
	case OP_MINUS, OP_SLASH, OP_ASTERISK:
		if leftType != NUM_TYPE {
			p.appendErrorForToken("'"+op.String()+"' takes num type, found "+leftType.Format(), tok)
		}
	case OP_LT, OP_GT, OP_LTEQ, OP_GTEQ:
		if leftType != NUM_TYPE && leftType != STRING_TYPE {
			p.appendErrorForToken("'"+op.String()+"' takes num or string type, found "+leftType.Format(), tok)
		}
	case OP_AND, OP_OR:
		if leftType != BOOL_TYPE {
			p.appendErrorForToken("'"+op.String()+"' takes bool type, found "+leftType.Format(), tok)
		}
	}
}

func (p *Parser) parseLiteral(scope *scope) Node {
	tok := p.cur
	tt := tok.TokenType()
	switch tt {
	case lexer.STRING_LIT:
		p.advance()
		return &StringLiteral{Token: tok, Value: tok.Literal}
	case lexer.NUM_LIT:
		p.advance()
		val, err := strconv.ParseFloat(tok.Literal, 64)
		if err != nil {
			p.appendError(err.Error())
			return nil
		}
		return &NumLiteral{Token: tok, Value: val}
	case lexer.TRUE, lexer.FALSE:
		p.advance()
		return &Bool{Token: tok, Value: tt == lexer.TRUE}
	case lexer.LBRACKET:
		return p.parseArrayLiteral(scope)
	case lexer.LCURLY:
		return p.parseMapLiteral(scope)
	}
	return nil
}

func (p *Parser) parseArrayLiteral(scope *scope) Node {
	tok := p.cur
	p.advance()     // advance past [
	p.advanceIfWS() // allow whitespace after `[`, eg [ 1 2 3 ]
	elements := p.parseExprList(scope)

	if elements == nil {
		return nil // previous error
	}
	if !p.assertToken(lexer.RBRACKET) {
		return nil
	}
	p.advance() // advance past ]
	if len(elements) == 0 {
		return &ArrayLiteral{Token: tok, T: GENERIC_ARRAY}
	}
	types := make([]*Type, len(elements))
	for i, e := range elements {
		types[i] = e.Type()
	}
	t := &Type{Name: ARRAY, Sub: p.combineTypes(types)}
	return &ArrayLiteral{Token: tok, Elements: elements, T: t}
}

func (p *Parser) parseExprList(scope *scope) []Node {
	list := []Node{}
	tt := p.cur.TokenType()
	for !p.isAtEOL() && tt != lexer.RPAREN && tt != lexer.RBRACKET {
		n := p.parseExprWSS(scope, LOWEST)
		if n == nil {
			return nil // previous error
		}
		list = append(list, n)
		p.advanceIfWS()
		tt = p.cur.TokenType()
	}
	return list
}

func (p *Parser) parseExprWSS(scope *scope, prec precedence) Node {
	p.pushWSS(true)
	defer p.popWSS()
	return p.parseExpr(scope, prec)
}

func (p *Parser) combineTypes(types []*Type) *Type {
	combinedT := types[0]
	for _, t := range types[1:] {
		if combinedT.Accepts(t) {
			continue
		}
		if t.Accepts(combinedT) {
			combinedT = t
			continue
		}
		return ANY_TYPE
	}
	return combinedT
}

func (p *Parser) parseMapLiteral(scope *scope) Node {
	p.pushWSS(false)
	defer p.popWSS()
	tok := p.cur
	p.advance() // advance past {
	pairs, order := p.parseMapPairs(scope)
	if pairs == nil {
		return nil // previous error
	}
	if !p.assertToken(lexer.RCURLY) {
		return nil
	}
	p.advance() // advance past }
	if len(pairs) == 0 {
		return &MapLiteral{Token: tok, T: GENERIC_MAP}
	}
	types := make([]*Type, 0, len(pairs))
	for _, n := range pairs {
		types = append(types, n.Type())
	}
	t := &Type{Name: MAP, Sub: p.combineTypes(types)}
	return &MapLiteral{Token: tok, Pairs: pairs, Order: order, T: t}
}

func (p *Parser) parseMapPairs(scope *scope) (map[string]Node, []string) {
	pairs := map[string]Node{}
	var order []string
	tt := p.cur.TokenType()

	for !p.isAtEOL() && tt != lexer.RCURLY {
		if tt != lexer.IDENT {
			p.appendError("expected map key, found " + p.cur.FormatDetails())
		}
		key := p.cur.Literal
		p.advance() // advance past key IDENT
		if _, ok := pairs[key]; ok {
			p.appendError("duplicated map key'" + key + "'")
			return nil, nil
		}
		p.assertToken(lexer.COLON)
		p.advance() // advance past COLON

		n := p.parseExprWSS(scope, LOWEST)
		if n == nil {
			return nil, nil // previous error
		}
		pairs[key] = n
		order = append(order, key)
		tt = p.cur.TokenType()
	}
	return pairs, order
}

// lookupVar looks up current token literal (IDENT) in scope.
// it assumes use, meaning reading of the variable, by marking the
// variable as used and hinting at using () around function calls.
// Do not use for writes, e.g. in left side of assignment.
func (p *Parser) lookupVar(scope *scope) Node {
	tok := p.cur
	name := p.cur.Literal
	p.advance()
	if v, ok := scope.get(name); ok {
		v.isUsed = true
		return v
	}
	if _, ok := p.funcs[name]; ok {
		p.appendErrorForToken("function call must be parenthesized: ("+name+" ...)", tok)
		return nil
	}
	p.appendErrorForToken("unknown variable name '"+name+"'", tok)
	return nil
}
