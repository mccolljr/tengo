package compiler_test

import (
	"testing"

	"github.com/d5/tengo/compiler"
)

func TestClass(t *testing.T) {
	expect(t, `class T {}`, bytecode(
		concat(
			compiler.MakeInstruction(compiler.OpGetBuiltin, 29),
			compiler.MakeInstruction(compiler.OpConstant, 0),
			compiler.MakeInstruction(compiler.OpMap, 0),
			compiler.MakeInstruction(compiler.OpCall, 2),
			compiler.MakeInstruction(compiler.OpSetGlobal, 0),
			compiler.MakeInstruction(compiler.OpSuspend),
		),
		objectsArray(
			stringObject("T"),
		),
	))

	expect(t, `class T {}; class U: T {}`, bytecode(
		concat(
			compiler.MakeInstruction(compiler.OpGetBuiltin, 29),
			compiler.MakeInstruction(compiler.OpConstant, 0),
			compiler.MakeInstruction(compiler.OpMap, 0),
			compiler.MakeInstruction(compiler.OpCall, 2),
			compiler.MakeInstruction(compiler.OpSetGlobal, 0),
			compiler.MakeInstruction(compiler.OpGetBuiltin, 29),
			compiler.MakeInstruction(compiler.OpGetGlobal, 0),
			compiler.MakeInstruction(compiler.OpConstant, 1),
			compiler.MakeInstruction(compiler.OpMap, 0),
			compiler.MakeInstruction(compiler.OpCall, 3),
			compiler.MakeInstruction(compiler.OpSetGlobal, 1),
			compiler.MakeInstruction(compiler.OpSuspend),
		),
		objectsArray(
			stringObject("T"),
			stringObject("U"),
		),
	))
}
