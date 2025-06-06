package parser

import (
	"fmt"
	"strconv"

	"github.com/inkbytefo/go-minus/internal/ast"
	"github.com/inkbytefo/go-minus/internal/token"
)

// parseExpression, bir ifadeyi ayrıştırır.
func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

// noPrefixParseFnError, bir prefix ayrıştırma fonksiyonu bulunamadığında bir hata ekler.
func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	msg := fmt.Sprintf("Satır %d, Sütun %d: %s için prefix ayrıştırma fonksiyonu bulunamadı",
		p.curToken.Line, p.curToken.Column, t)
	p.errors = append(p.errors, msg)
}

// parseIdentifier, bir tanımlayıcıyı ayrıştırır.
func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

// parseIntegerLiteral, bir tamsayı değişmez değerini ayrıştırır.
func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("Satır %d, Sütun %d: %q bir tamsayıya dönüştürülemedi",
			p.curToken.Line, p.curToken.Column, p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}

	lit.Value = value
	return lit
}

// parseFloatLiteral, bir ondalık sayı değişmez değerini ayrıştırır.
func (p *Parser) parseFloatLiteral() ast.Expression {
	lit := &ast.FloatLiteral{Token: p.curToken}

	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		msg := fmt.Sprintf("Satır %d, Sütun %d: %q bir ondalık sayıya dönüştürülemedi",
			p.curToken.Line, p.curToken.Column, p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}

	lit.Value = value
	return lit
}

// parseStringLiteral, bir dize değişmez değerini ayrıştırır.
func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

// parseCharLiteral, bir karakter değişmez değerini ayrıştırır.
func (p *Parser) parseCharLiteral() ast.Expression {
	if len(p.curToken.Literal) != 1 {
		msg := fmt.Sprintf("Satır %d, Sütun %d: %q bir karakter değil",
			p.curToken.Line, p.curToken.Column, p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}
	return &ast.CharLiteral{Token: p.curToken, Value: rune(p.curToken.Literal[0])}
}

// parseBooleanLiteral, bir boolean değişmez değerini ayrıştırır.
func (p *Parser) parseBooleanLiteral() ast.Expression {
	return &ast.BooleanLiteral{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

// parseNullLiteral, bir null değişmez değerini ayrıştırır.
func (p *Parser) parseNullLiteral() ast.Expression {
	return &ast.NullLiteral{Token: p.curToken}
}

// parsePrefixExpression, bir önek ifadesini ayrıştırır.
func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()

	expression.Right = p.parseExpression(PREFIX)

	return expression
}

// parseInfixExpression, bir araek ifadesini ayrıştırır.
func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

// parseGroupedExpression, parantez içindeki bir ifadeyi ayrıştırır.
func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
}

// parseArrayLiteral, bir dizi değişmez değerini ayrıştırır.
func (p *Parser) parseArrayLiteral() ast.Expression {
	tok := p.curToken

	// Check if this is a typed array literal: [5]int{1,2,3} or []int{1,2,3}
	// or a simple array literal: [1,2,3]

	// Look ahead to see if this is a typed array literal
	if p.isTypedArrayLiteral() {
		return p.parseTypedArrayLiteral()
	}

	// Simple array literal: [1, 2, 3]
	array := &ast.ArrayLiteral{Token: tok}
	array.Elements = p.parseExpressionList(token.RBRACKET)
	return array
}

// parseArrayType, bir array type'ını ayrıştırır.
func (p *Parser) parseArrayType() ast.Expression {
	tok := p.curToken // LBRACKET token

	var size ast.Expression

	// Check if this is a slice type []Type or array type [size]Type
	if !p.peekTokenIs(token.RBRACKET) {
		// Array type with size: [5]int
		p.nextToken()
		size = p.parseExpression(LOWEST)
	}
	// else: slice type []int

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	// Parse element type
	if !p.peekTokenIs(token.IDENT) && !p.peekTokenIs(token.INT) && !p.peekTokenIs(token.STRING) {
		// Not followed by a type, this might be an array literal
		return nil
	}

	p.nextToken()
	elementType := p.parseExpression(LOWEST)

	return &ast.ArrayType{
		Token:       tok,
		Size:        size, // nil for slices
		ElementType: elementType,
	}
}

// parseHashLiteral, bir hash değişmez değerini ayrıştırır.
func (p *Parser) parseHashLiteral() ast.Expression {
	hash := &ast.HashLiteral{Token: p.curToken}
	hash.Pairs = make(map[ast.Expression]ast.Expression)

	for !p.peekTokenIs(token.RBRACE) && !p.peekTokenIs(token.EOF) {
		p.nextToken()
		key := p.parseExpression(LOWEST)

		if !p.expectPeek(token.COLON) {
			return nil
		}

		p.nextToken()
		value := p.parseExpression(LOWEST)

		hash.Pairs[key] = value

		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			return nil
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return hash
}

// parseIndexExpression, bir dizin ifadesini ayrıştırır.
func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: p.curToken, Left: left}

	p.nextToken()
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	return exp
}

// parseMemberExpression, bir üye erişim ifadesini ayrıştırır.
func (p *Parser) parseMemberExpression(object ast.Expression) ast.Expression {
	exp := &ast.MemberExpression{
		Token:  p.curToken,
		Object: object,
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	exp.Member = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	return exp
}

// parseAssignExpression, bir atama ifadesini ayrıştırır.
func (p *Parser) parseAssignExpression(left ast.Expression) ast.Expression {
	exp := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	exp.Right = p.parseExpression(precedence)

	return exp
}

// parseShortVarDeclExpression, bir kısa değişken tanımlama ifadesini ayrıştırır.
func (p *Parser) parseShortVarDeclExpression(left ast.Expression) ast.Expression {
	exp := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	exp.Right = p.parseExpression(precedence)

	return exp
}

// parseExpressionList, bir ifade listesini ayrıştırır.
func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	list := []ast.Expression{}

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

// parsePostfixExpression, bir postfix ifadesini ayrıştırır (++ ve -- operatörleri için).
func (p *Parser) parsePostfixExpression(left ast.Expression) ast.Expression {
	return &ast.PostfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}
}

// parseExpressionUntil, belirtilen token'a kadar ifadeyi ayrıştırır.
func (p *Parser) parseExpressionUntil(precedence int, until token.TokenType) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(until) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

// isTypedArrayLiteral, mevcut pozisyonun typed array literal olup olmadığını kontrol eder.
func (p *Parser) isTypedArrayLiteral() bool {
	// Basit kontrol: şimdilik sadece simple array literal'ları destekleyelim
	// Typed array literal desteği daha sonra eklenebilir
	return false
}

// parseTypedArrayLiteral, typed array literal'ı ayrıştırır: [5]int{1,2,3}
func (p *Parser) parseTypedArrayLiteral() ast.Expression {
	tok := p.curToken // LBRACKET token

	// Parse array type first
	arrayType := p.parseArrayType()
	if arrayType == nil {
		return nil
	}

	// Expect {
	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	// Parse elements
	elements := p.parseExpressionList(token.RBRACE)

	// Create a typed array literal (we'll use ArrayLiteral with type info)
	arrayLit := &ast.ArrayLiteral{
		Token:    tok,
		Elements: elements,
	}

	// Store type information (we might need a new AST node for this)
	// For now, we'll handle this in semantic analysis

	return arrayLit
}
