package runtime

import (
	"github.com/d5/tengo/compiler"
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

func (v *VM) openStackFrame(fn *objects.CompiledFunction,
	numArgs int, vars []*objects.ObjectPtr) {

	// update call frame
	v.curFrame.ip = v.ip // store current ip before call
	v.curFrame = &v.frames[v.framesIndex]
	v.curFrame.fn = fn
	v.curFrame.freeVars = vars
	v.curFrame.basePointer = v.sp - numArgs
	v.curInsts = fn.Instructions
	v.ip = -1
	v.framesIndex++
	v.sp = v.sp - numArgs + fn.NumLocals
}

func (v *VM) closeStackFrame() {
	v.framesIndex--
	v.curFrame = &v.frames[v.framesIndex-1]
	v.curInsts = v.curFrame.fn.Instructions
	v.ip = v.curFrame.ip
	v.sp = v.frames[v.framesIndex].basePointer
}

func (v *VM) callObject(what objects.Object,
	vars []*objects.ObjectPtr, args []objects.Object) {

	v.stack[v.sp] = what
	v.sp++
	numArgs := len(args)
	for _, arg := range args {
		v.stack[v.sp] = arg
		v.sp++
	}

	fn := &objects.CompiledFunction{
		Instructions: compiler.MakeInstruction(compiler.OpCall, numArgs),
	}

	v.openStackFrame(fn, numArgs, vars)

}
