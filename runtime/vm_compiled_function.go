package runtime

import (
	"bytes"

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

func (v *VM) callObject(what objects.Object, args []objects.Object) (objects.Object, error) {

	numArgs := len(args)

	fn := &objects.CompiledFunction{
		Instructions: bytes.Join([][]byte{
			compiler.MakeInstruction(compiler.OpCall, numArgs),
			compiler.MakeInstruction(compiler.OpBreak),
		}, []byte{}),
		SourceMap: map[int]source.Pos{},
	}

	// update call frame
	v.curFrame.ip = v.ip // store current ip before call
	v.curFrame = &v.frames[v.framesIndex]
	v.curFrame.fn = fn
	v.curFrame.freeVars = nil
	v.curFrame.basePointer = v.sp
	v.curInsts = fn.Instructions
	v.ip = -1
	v.framesIndex++

	v.stack[v.sp] = what
	v.sp++
	for _, arg := range args {
		v.stack[v.sp] = arg
		v.sp++
	}

	v.run()
	if v.err != nil {
		return nil, v.err
	}

	retVal := v.stack[v.sp-1]

	v.framesIndex--
	v.curFrame = &v.frames[v.framesIndex-1]
	v.curInsts = v.curFrame.fn.Instructions
	v.ip = v.curFrame.ip
	v.sp = v.frames[v.framesIndex].basePointer

	return retVal, nil
}
