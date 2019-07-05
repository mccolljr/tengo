package ast

import (
	"fmt"

	"github.com/d5/tengo/compiler/source"
)

// ClassStmt represents a class statement.
type ClassStmt struct {
	ClassPos source.Pos
	Name     *Ident
	Extends  *Ident
	Body     *MapLit
}

func (s *ClassStmt) stmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *ClassStmt) Pos() source.Pos {
	return s.ClassPos
}

// End returns the position of first character immediately after the node.
func (s *ClassStmt) End() source.Pos {
	return s.Body.End()
}

func (s *ClassStmt) String() string {
	if s.Extends != nil {
		return fmt.Sprintf("class %s: %s %s", s.Name.Name, s.Extends.Name, s.Body)
	}

	return fmt.Sprintf("class %s %s", s.Name.Name, s.Body)
}
