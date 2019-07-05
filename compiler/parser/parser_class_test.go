package parser_test

import (
	"testing"

	"github.com/d5/tengo/compiler/ast"
)

func TestClass(t *testing.T) {
	expectError(t, "class := func()")
	expectError(t, "class()")
	expectError(t, `x := class("x", {})`)
	expectError(t, `base := class("base", {}); x := class(base, "x", {})`)

	expectString(t, "class X {}", "class X {}")
	expectString(t, "class X: Y {}", "class X: Y {}")

	expect(t, "class X {}", func(p pfn) []ast.Stmt {
		return stmts(
			classStmt(
				p(1, 1),
				ident("X", p(1, 7)), nil,
				mapLit(p(1, 9), p(1, 10))))
	})

	expect(t, "class X: Y {}", func(p pfn) []ast.Stmt {
		return stmts(
			classStmt(
				p(1, 1),
				ident("X", p(1, 7)),
				ident("Y", p(1, 10)),
				mapLit(p(1, 12), p(1, 13))))
	})
}
