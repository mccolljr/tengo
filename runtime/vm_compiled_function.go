package runtime

import (
	"github.com/d5/tengo/compiler/source"
	"github.com/d5/tengo/objects"
)

type compiledFunction struct {
	Instructions  []byte
	NumLocals     int
	NumParameters int
	VarArgs       bool
	SourceMap     map[int]source.Pos
	vmCall        func(objects.Object, []objects.Object) (objects.Object, error)
}
