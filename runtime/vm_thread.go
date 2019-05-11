package runtime

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/d5/tengo/compiler"
	"github.com/d5/tengo/compiler/source"
	"github.com/d5/tengo/compiler/token"
	"github.com/d5/tengo/objects"
)

// Thread ...
type Thread struct {
	vm       *VM
	parentID uint64
	id       uint64

	isMain      bool
	stack       [StackSize]objects.Object
	sp          int
	frames      [MaxFrames]Frame
	framesIndex int
	curFrame    *Frame
	curInsts    []byte
	ip          int
	err         error
}

func (v *Thread) call(fn objects.Object, args []objects.Object) (retVal objects.Object, retErr error) {
	numArgs := len(args)

	// check for stack overflow
	if v.sp+numArgs >= StackSize {
		return nil, ErrStackOverflow
	}

	// create a micro-function to handle the call
	callee := &objects.CompiledFunction{
		Instructions: []byte{
			/* CALL		numArgs */ compiler.OpCall, byte(numArgs),
			/* SUSPEND			*/ compiler.OpSuspend,
		},
		SourceMap: map[int]source.Pos{},
	}

	// create a call frame for our micro-function
	v.curFrame.ip = v.ip // store current ip before call
	v.curFrame = &(v.frames[v.framesIndex])
	v.curFrame.fn = callee
	v.curFrame.freeVars = nil
	v.curFrame.basePointer = v.sp
	v.curInsts = callee.Instructions
	v.ip = -1
	v.framesIndex++

	// set up the stack for the call in the micro-function
	spStart := v.sp
	spEnd := spStart + numArgs + 1
	v.stack[v.sp] = fn
	v.sp++
	argStart := v.sp
	copy(v.stack[argStart:spEnd], args)
	v.sp += numArgs

	// resume execution
	v.run()

	// capture error or return value
	if v.err != nil {
		return nil, v.err
	}
	// no error, get return
	retVal = v.stack[v.sp-1]

	// leave frame
	v.framesIndex--
	v.curFrame = &v.frames[v.framesIndex-1]
	v.curInsts = v.curFrame.fn.Instructions
	v.ip = v.curFrame.ip
	v.sp = v.frames[v.framesIndex].basePointer

	return retVal, retErr
}

func (v *Thread) execute() {
	v.vm.acquire(v.id)
	if v.vm.currentThread != v {
		panic("uh oh")
	}
	v.run()
	if !v.isMain {
		if v.err != nil {
			v.vm.threadResults[v.id] = &objects.Error{
				Value: &objects.String{
					Value: v.err.Error(),
				},
			}
		} else {
			ret := v.stack[0]
			if ret == nil {
				ret = objects.UndefinedValue
			}
			v.vm.threadResults[v.id] = ret
		}
		delete(v.vm.threads, v.id)
		v.vm.threadPool <- v
	}
	v.vm.release()
}

func (v *Thread) run() {
	defer func() {
		if r := recover(); r != nil {
			if v.sp >= StackSize || v.framesIndex >= MaxFrames {
				v.err = ErrStackOverflow
				return
			}

			if v.ip < len(v.curInsts)-1 {
				if err, ok := r.(error); ok {
					v.err = err
				} else {
					v.err = fmt.Errorf("%d panic: %v", v.id, r)
				}
			}
		}
	}()

	for atomic.LoadInt64(&v.vm.aborting) == 0 {
		v.ip++

		switch v.curInsts[v.ip] {
		case compiler.OpSuspend:
			return

		case compiler.OpSleep:
			val := v.stack[v.sp-1]
			// v.sp--
			intVal, _ := objects.ToInt64(val)
			v.vm.release()
			time.Sleep(time.Duration(intVal) * time.Millisecond)
			v.vm.acquire(v.id)

		case compiler.OpSpawn:
			val := v.stack[v.sp-1]
			// v.sp--
			fn, _ := val.(*objects.CompiledFunction)
			if fn == nil {
				v.err = fmt.Errorf("cannot spawn thread with non-func %s", val.TypeName())
				return
			}

			if fn.NumParameters != 0 {
				v.err = errors.New("thread func must take no parameters")
				return
			}

			t := v.vm.createThread(fn)
			v.stack[v.sp] = &objects.Int{Value: int64(t.id)}
			v.sp++
			go t.execute()

		case compiler.OpWait:
			val := v.stack[v.sp-1]
			// v.sp--
			intVal, _ := objects.ToInt64(val)
			threadID := uint64(intVal)

			if threadID == MainThreadID {
				v.err = errors.New("invalid wait: cannot wait on main thread")
				return
			}

			if t, _ := v.vm.threads[threadID]; t == nil {
				v.err = fmt.Errorf("invalid wait: no such thread %d", threadID)
				return
			}

			r := v.vm.threadResults[threadID]
			for r == nil {
				v.vm.release()
				time.Sleep(100 * time.Millisecond)
				v.vm.acquire(v.id)
				r = v.vm.threadResults[threadID]
			}
			delete(v.vm.threadResults, threadID)

			v.stack[v.sp] = r
			v.sp++

		case compiler.OpConstant:
			v.ip += 2
			cidx := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8

			v.stack[v.sp] = v.vm.constants[cidx]
			v.sp++

		case compiler.OpNull:
			v.stack[v.sp] = objects.UndefinedValue
			v.sp++

		case compiler.OpBinaryOp:
			v.ip++
			right := v.stack[v.sp-1]
			left := v.stack[v.sp-2]

			tok := token.Token(v.curInsts[v.ip])
			res, e := left.BinaryOp(tok, right)
			if e != nil {
				v.sp -= 2

				if e == objects.ErrInvalidOperator {
					v.err = fmt.Errorf("invalid operation: %s %s %s",
						left.TypeName(), tok.String(), right.TypeName())
					return
				}

				v.err = e
				return
			}

			v.vm.allocs--
			if v.vm.allocs == 0 {
				v.vm.err = ErrObjectAllocLimit
				return
			}

			v.stack[v.sp-2] = res
			v.sp--

		case compiler.OpEqual:
			right := v.stack[v.sp-1]
			left := v.stack[v.sp-2]
			v.sp -= 2

			if left.Equals(right) {
				v.stack[v.sp] = objects.TrueValue
			} else {
				v.stack[v.sp] = objects.FalseValue
			}
			v.sp++

		case compiler.OpNotEqual:
			right := v.stack[v.sp-1]
			left := v.stack[v.sp-2]
			v.sp -= 2

			if left.Equals(right) {
				v.stack[v.sp] = objects.FalseValue
			} else {
				v.stack[v.sp] = objects.TrueValue
			}
			v.sp++

		case compiler.OpPop:
			v.sp--

		case compiler.OpTrue:
			v.stack[v.sp] = objects.TrueValue
			v.sp++

		case compiler.OpFalse:
			v.stack[v.sp] = objects.FalseValue
			v.sp++

		case compiler.OpLNot:
			operand := v.stack[v.sp-1]
			v.sp--

			if operand.IsFalsy() {
				v.stack[v.sp] = objects.TrueValue
			} else {
				v.stack[v.sp] = objects.FalseValue
			}
			v.sp++

		case compiler.OpBComplement:
			operand := v.stack[v.sp-1]
			v.sp--

			switch x := operand.(type) {
			case *objects.Int:
				var res objects.Object = &objects.Int{Value: ^x.Value}

				v.vm.allocs--
				if v.vm.allocs == 0 {
					v.vm.err = ErrObjectAllocLimit
					return
				}

				v.stack[v.sp] = res
				v.sp++
			default:
				v.err = fmt.Errorf("invalid operation: ^%s", operand.TypeName())
				return
			}

		case compiler.OpMinus:
			operand := v.stack[v.sp-1]
			v.sp--

			switch x := operand.(type) {
			case *objects.Int:
				var res objects.Object = &objects.Int{Value: -x.Value}

				v.vm.allocs--
				if v.vm.allocs == 0 {
					v.vm.err = ErrObjectAllocLimit
					return
				}

				v.stack[v.sp] = res
				v.sp++
			case *objects.Float:
				var res objects.Object = &objects.Float{Value: -x.Value}

				v.vm.allocs--
				if v.vm.allocs == 0 {
					v.vm.err = ErrObjectAllocLimit
					return
				}

				v.stack[v.sp] = res
				v.sp++
			default:
				v.err = fmt.Errorf("invalid operation: -%s", operand.TypeName())
				return
			}

		case compiler.OpJumpFalsy:
			v.ip += 2
			v.sp--
			if v.stack[v.sp].IsFalsy() {
				pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
				v.ip = pos - 1
			}

		case compiler.OpAndJump:
			v.ip += 2

			if v.stack[v.sp-1].IsFalsy() {
				pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
				v.ip = pos - 1
			} else {
				v.sp--
			}

		case compiler.OpOrJump:
			v.ip += 2

			if v.stack[v.sp-1].IsFalsy() {
				v.sp--
			} else {
				pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
				v.ip = pos - 1
			}

		case compiler.OpJump:
			pos := int(v.curInsts[v.ip+2]) | int(v.curInsts[v.ip+1])<<8
			v.ip = pos - 1

		case compiler.OpSetGlobal:
			v.ip += 2
			v.sp--

			globalIndex := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
			v.vm.globals[globalIndex] = v.stack[v.sp]

		case compiler.OpSetSelGlobal:
			v.ip += 3
			globalIndex := int(v.curInsts[v.ip-1]) | int(v.curInsts[v.ip-2])<<8
			numSelectors := int(v.curInsts[v.ip])

			// selectors and RHS value
			selectors := make([]objects.Object, numSelectors)
			for i := 0; i < numSelectors; i++ {
				selectors[i] = v.stack[v.sp-numSelectors+i]
			}

			val := v.stack[v.sp-numSelectors-1]
			v.sp -= numSelectors + 1

			if e := indexAssign(v.vm.globals[globalIndex], val, selectors); e != nil {
				v.err = e
				return
			}

		case compiler.OpGetGlobal:
			v.ip += 2
			globalIndex := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8

			val := v.vm.globals[globalIndex]

			v.stack[v.sp] = val
			v.sp++

		case compiler.OpArray:
			v.ip += 2
			numElements := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8

			var elements []objects.Object
			for i := v.sp - numElements; i < v.sp; i++ {
				elt := v.stack[i]
				if spread, ok := elt.(*objects.Spread); ok {
					elements = append(elements, spread.Values...)
				} else {
					elements = append(elements, elt)
				}
			}

			v.sp -= numElements

			var arr objects.Object = &objects.Array{Value: elements}

			v.vm.allocs--
			if v.vm.allocs == 0 {
				v.vm.err = ErrObjectAllocLimit
				return
			}

			v.stack[v.sp] = arr
			v.sp++

		case compiler.OpMap:
			v.ip += 2
			numElements := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8

			kv := make(map[string]objects.Object)
			for i := v.sp - numElements; i < v.sp; i += 2 {
				key := v.stack[i]
				value := v.stack[i+1]
				kv[key.(*objects.String).Value] = value
			}
			v.sp -= numElements

			var m objects.Object = &objects.Map{Value: kv}

			v.vm.allocs--
			if v.vm.allocs == 0 {
				v.vm.err = ErrObjectAllocLimit
				return
			}

			v.stack[v.sp] = m
			v.sp++

		case compiler.OpError:
			value := v.stack[v.sp-1]

			var e objects.Object = &objects.Error{
				Value: value,
			}

			v.vm.allocs--
			if v.vm.allocs == 0 {
				v.vm.err = ErrObjectAllocLimit
				return
			}

			v.stack[v.sp-1] = e

		case compiler.OpImmutable:
			value := v.stack[v.sp-1]

			switch value := value.(type) {
			case *objects.Array:
				var immutableArray objects.Object = &objects.ImmutableArray{
					Value: value.Value,
				}

				v.vm.allocs--
				if v.vm.allocs == 0 {
					v.vm.err = ErrObjectAllocLimit
					return
				}

				v.stack[v.sp-1] = immutableArray
			case *objects.Map:
				var immutableMap objects.Object = &objects.ImmutableMap{
					Value: value.Value,
				}

				v.vm.allocs--
				if v.vm.allocs == 0 {
					v.vm.err = ErrObjectAllocLimit
					return
				}

				v.stack[v.sp-1] = immutableMap
			}

		case compiler.OpIndex:
			index := v.stack[v.sp-1]
			left := v.stack[v.sp-2]
			v.sp -= 2

			val, err := left.IndexGet(index)
			if err != nil {
				if err == objects.ErrNotIndexable {
					v.err = fmt.Errorf("not indexable: %s", index.TypeName())
					return
				}

				if err == objects.ErrInvalidIndexType {
					v.err = fmt.Errorf("invalid index type: %s", index.TypeName())
					return
				}

				v.err = err
				return
			}

			if val == nil {
				val = objects.UndefinedValue
			}

			v.stack[v.sp] = val
			v.sp++

		case compiler.OpSliceIndex:
			high := v.stack[v.sp-1]
			low := v.stack[v.sp-2]
			left := v.stack[v.sp-3]
			v.sp -= 3

			var lowIdx int64
			if low != objects.UndefinedValue {
				if low, ok := low.(*objects.Int); ok {
					lowIdx = low.Value
				} else {
					v.err = fmt.Errorf("invalid slice index type: %s", low.TypeName())
					return
				}
			}

			switch left := left.(type) {
			case *objects.Array:
				numElements := int64(len(left.Value))
				var highIdx int64
				if high == objects.UndefinedValue {
					highIdx = numElements
				} else if high, ok := high.(*objects.Int); ok {
					highIdx = high.Value
				} else {
					v.err = fmt.Errorf("invalid slice index type: %s", high.TypeName())
					return
				}

				if lowIdx > highIdx {
					v.err = fmt.Errorf("invalid slice index: %d > %d", lowIdx, highIdx)
					return
				}

				if lowIdx < 0 {
					lowIdx = 0
				} else if lowIdx > numElements {
					lowIdx = numElements
				}

				if highIdx < 0 {
					highIdx = 0
				} else if highIdx > numElements {
					highIdx = numElements
				}

				var val objects.Object = &objects.Array{Value: left.Value[lowIdx:highIdx]}

				v.vm.allocs--
				if v.vm.allocs == 0 {
					v.vm.err = ErrObjectAllocLimit
					return
				}

				v.stack[v.sp] = val
				v.sp++

			case *objects.ImmutableArray:
				numElements := int64(len(left.Value))
				var highIdx int64
				if high == objects.UndefinedValue {
					highIdx = numElements
				} else if high, ok := high.(*objects.Int); ok {
					highIdx = high.Value
				} else {
					v.err = fmt.Errorf("invalid slice index type: %s", high.TypeName())
					return
				}

				if lowIdx > highIdx {
					v.err = fmt.Errorf("invalid slice index: %d > %d", lowIdx, highIdx)
					return
				}

				if lowIdx < 0 {
					lowIdx = 0
				} else if lowIdx > numElements {
					lowIdx = numElements
				}

				if highIdx < 0 {
					highIdx = 0
				} else if highIdx > numElements {
					highIdx = numElements
				}

				var val objects.Object = &objects.Array{Value: left.Value[lowIdx:highIdx]}

				v.vm.allocs--
				if v.vm.allocs == 0 {
					v.vm.err = ErrObjectAllocLimit
					return
				}

				v.stack[v.sp] = val
				v.sp++

			case *objects.String:
				numElements := int64(len(left.Value))
				var highIdx int64
				if high == objects.UndefinedValue {
					highIdx = numElements
				} else if high, ok := high.(*objects.Int); ok {
					highIdx = high.Value
				} else {
					v.err = fmt.Errorf("invalid slice index type: %s", high.TypeName())
					return
				}

				if lowIdx > highIdx {
					v.err = fmt.Errorf("invalid slice index: %d > %d", lowIdx, highIdx)
					return
				}

				if lowIdx < 0 {
					lowIdx = 0
				} else if lowIdx > numElements {
					lowIdx = numElements
				}

				if highIdx < 0 {
					highIdx = 0
				} else if highIdx > numElements {
					highIdx = numElements
				}

				var val objects.Object = &objects.String{Value: left.Value[lowIdx:highIdx]}

				v.vm.allocs--
				if v.vm.allocs == 0 {
					v.vm.err = ErrObjectAllocLimit
					return
				}

				v.stack[v.sp] = val
				v.sp++

			case *objects.Bytes:
				numElements := int64(len(left.Value))
				var highIdx int64
				if high == objects.UndefinedValue {
					highIdx = numElements
				} else if high, ok := high.(*objects.Int); ok {
					highIdx = high.Value
				} else {
					v.err = fmt.Errorf("invalid slice index type: %s", high.TypeName())
					return
				}

				if lowIdx > highIdx {
					v.err = fmt.Errorf("invalid slice index: %d > %d", lowIdx, highIdx)
					return
				}

				if lowIdx < 0 {
					lowIdx = 0
				} else if lowIdx > numElements {
					lowIdx = numElements
				}

				if highIdx < 0 {
					highIdx = 0
				} else if highIdx > numElements {
					highIdx = numElements
				}

				var val objects.Object = &objects.Bytes{Value: left.Value[lowIdx:highIdx]}

				v.vm.allocs--
				if v.vm.allocs == 0 {
					v.vm.err = ErrObjectAllocLimit
					return
				}

				v.stack[v.sp] = val
				v.sp++
			}

		case compiler.OpSpread:
			spreadSP := v.sp - 1
			target := v.stack[spreadSP]
			if !target.CanSpread() {
				v.err = fmt.Errorf("cannot spread value of type %s", target.TypeName())
				return
			}

			v.stack[spreadSP] = &objects.Spread{
				Values: target.Spread(),
			}

		case compiler.OpCall:
			numArgs := int(v.curInsts[v.ip+1])
			v.ip++
			spBase := v.sp - 1 - numArgs
			value := v.stack[spBase]

			if !value.CanCall() {
				v.err = fmt.Errorf("not callable: %s", value.TypeName())
				return
			}

			if numArgs > 0 {
				i := v.sp - 1
				arg := v.stack[i]
				if spread, ok := arg.(*objects.Spread); ok {
					list := spread.Values
					numSpreadValues := len(list)
					if v.sp+numSpreadValues >= StackSize {
						v.err = ErrStackOverflow
						return
					}
					ebStart, ebEnd := i, i+numSpreadValues
					rxStart, rxEnd := i+1, spBase+numArgs+1
					rmStart, rmEnd := i+numSpreadValues, spBase+numArgs+numSpreadValues
					copy(v.stack[rmStart:rmEnd], v.stack[rxStart:rxEnd])
					copy(v.stack[ebStart:ebEnd], list)
					numArgs += numSpreadValues - 1
					v.sp += numSpreadValues - 1
				}
			}

			if callee, ok := value.(*objects.CompiledFunction); ok {
				if callee.VarArgs {
					// if the closure is variadic,
					// roll up all variadic parameters into an array
					realArgs := callee.NumParameters - 1
					varArgs := numArgs - realArgs
					if varArgs >= 0 {
						numArgs = realArgs + 1
						args := make([]objects.Object, varArgs)
						spStart := v.sp - varArgs
						for i := spStart; i < v.sp; i++ {
							args[i-spStart] = v.stack[i]
						}
						v.stack[spStart] = &objects.Array{Value: args}
						v.sp = spStart + 1
					}
				}

				if numArgs != callee.NumParameters {
					if callee.VarArgs {
						v.err = fmt.Errorf("wrong number of arguments: want>=%d, got=%d",
							callee.NumParameters-1, numArgs)
					} else {
						v.err = fmt.Errorf("wrong number of arguments: want=%d, got=%d",
							callee.NumParameters, numArgs)
					}
					return
				}

				// test if it's tail-call
				if callee == v.curFrame.fn { // recursion
					nextOp := v.curInsts[v.ip+1]
					if nextOp == compiler.OpReturn ||
						(nextOp == compiler.OpPop && compiler.OpReturn == v.curInsts[v.ip+2]) {
						for p := 0; p < numArgs; p++ {
							v.stack[v.curFrame.basePointer+p] = v.stack[v.sp-numArgs+p]
						}
						v.sp -= numArgs + 1
						v.ip = -1 // reset IP to beginning of the frame
						continue
					}
				}

				// update call frame
				v.curFrame.ip = v.ip // store current ip before call
				v.curFrame = &(v.frames[v.framesIndex])
				v.curFrame.fn = callee
				v.curFrame.freeVars = callee.Free
				v.curFrame.basePointer = v.sp - numArgs
				v.curInsts = callee.Instructions
				v.ip = -1
				v.framesIndex++
				v.sp = v.sp - numArgs + callee.NumLocals
			} else {
				var args []objects.Object
				args = append(args, v.stack[v.sp-numArgs:v.sp]...)

				ret, e := value.Call(v.vm, args...)
				v.sp -= numArgs + 1

				// runtime error
				if e != nil {
					if e == objects.ErrWrongNumArguments {
						v.err = fmt.Errorf("wrong number of arguments in call to '%s'",
							value.TypeName())
						return
					}

					if e, ok := e.(objects.ErrInvalidArgumentType); ok {
						v.err = fmt.Errorf("invalid type for argument '%s' in call to '%s': expected %s, found %s",
							e.Name, value.TypeName(), e.Expected, e.Found)
						return
					}

					v.err = e
					return
				}

				// nil return -> undefined
				if ret == nil {
					ret = objects.UndefinedValue
				}

				v.vm.allocs--
				if v.vm.allocs == 0 {
					v.vm.err = ErrObjectAllocLimit
					return
				}

				v.stack[v.sp] = ret
				v.sp++
			}

		case compiler.OpReturn:
			v.ip++
			var retVal objects.Object
			if int(v.curInsts[v.ip]) == 1 {
				retVal = v.stack[v.sp-1]
			} else {
				retVal = objects.UndefinedValue
			}
			//v.sp--

			v.framesIndex--
			v.curFrame = &v.frames[v.framesIndex-1]
			v.curInsts = v.curFrame.fn.Instructions
			v.ip = v.curFrame.ip

			//v.sp = lastFrame.basePointer - 1
			v.sp = v.frames[v.framesIndex].basePointer

			// skip stack overflow check because (newSP) <= (oldSP)
			v.stack[v.sp-1] = retVal
			//v.sp++

		case compiler.OpDefineLocal:
			v.ip++
			localIndex := int(v.curInsts[v.ip])

			sp := v.curFrame.basePointer + localIndex

			// local variables can be mutated by other actions
			// so always store the copy of popped value
			val := v.stack[v.sp-1]
			v.sp--

			v.stack[sp] = val

		case compiler.OpSetLocal:
			localIndex := int(v.curInsts[v.ip+1])
			v.ip++

			sp := v.curFrame.basePointer + localIndex

			// update pointee of v.stack[sp] instead of replacing the pointer itself.
			// this is needed because there can be free variables referencing the same local variables.
			val := v.stack[v.sp-1]
			v.sp--

			if obj, ok := v.stack[sp].(*objects.ObjectPtr); ok {
				*obj.Value = val
				val = obj
			}
			v.stack[sp] = val // also use a copy of popped value

		case compiler.OpSetSelLocal:
			localIndex := int(v.curInsts[v.ip+1])
			numSelectors := int(v.curInsts[v.ip+2])
			v.ip += 2

			// selectors and RHS value
			selectors := make([]objects.Object, numSelectors)
			for i := 0; i < numSelectors; i++ {
				selectors[i] = v.stack[v.sp-numSelectors+i]
			}

			val := v.stack[v.sp-numSelectors-1]
			v.sp -= numSelectors + 1

			dst := v.stack[v.curFrame.basePointer+localIndex]
			if obj, ok := dst.(*objects.ObjectPtr); ok {
				dst = *obj.Value
			}

			if e := indexAssign(dst, val, selectors); e != nil {
				v.err = e
				return
			}

		case compiler.OpGetLocal:
			v.ip++
			localIndex := int(v.curInsts[v.ip])

			val := v.stack[v.curFrame.basePointer+localIndex]

			if obj, ok := val.(*objects.ObjectPtr); ok {
				val = *obj.Value
			}

			v.stack[v.sp] = val
			v.sp++

		case compiler.OpGetBuiltin:
			v.ip++
			builtinIndex := int(v.curInsts[v.ip])

			v.stack[v.sp] = objects.Builtins[builtinIndex]
			v.sp++

		case compiler.OpClosure:
			v.ip += 3
			constIndex := int(v.curInsts[v.ip-1]) | int(v.curInsts[v.ip-2])<<8
			numFree := int(v.curInsts[v.ip])

			fn, ok := v.vm.constants[constIndex].(*objects.CompiledFunction)
			if !ok {
				v.err = fmt.Errorf("not function: %s", fn.TypeName())
				return
			}

			free := make([]*objects.ObjectPtr, numFree)
			for i := 0; i < numFree; i++ {
				switch freeVar := (v.stack[v.sp-numFree+i]).(type) {
				case *objects.ObjectPtr:
					free[i] = freeVar
				default:
					free[i] = &objects.ObjectPtr{Value: &v.stack[v.sp-numFree+i]}
				}
			}

			v.sp -= numFree

			cl := &objects.CompiledFunction{
				Instructions:  fn.Instructions,
				NumLocals:     fn.NumLocals,
				NumParameters: fn.NumParameters,
				VarArgs:       fn.VarArgs,
				Free:          free,
			}

			v.vm.allocs--
			if v.vm.allocs == 0 {
				v.vm.err = ErrObjectAllocLimit
				return
			}

			v.stack[v.sp] = cl
			v.sp++

		case compiler.OpGetFreePtr:
			v.ip++
			freeIndex := int(v.curInsts[v.ip])

			val := v.curFrame.freeVars[freeIndex]

			v.stack[v.sp] = val
			v.sp++

		case compiler.OpGetFree:
			v.ip++
			freeIndex := int(v.curInsts[v.ip])

			val := *v.curFrame.freeVars[freeIndex].Value

			v.stack[v.sp] = val
			v.sp++

		case compiler.OpSetFree:
			v.ip++
			freeIndex := int(v.curInsts[v.ip])

			*v.curFrame.freeVars[freeIndex].Value = v.stack[v.sp-1]

			v.sp--

		case compiler.OpGetLocalPtr:
			v.ip++
			localIndex := int(v.curInsts[v.ip])

			sp := v.curFrame.basePointer + localIndex
			val := v.stack[sp]

			var freeVar *objects.ObjectPtr
			if obj, ok := val.(*objects.ObjectPtr); ok {
				freeVar = obj
			} else {
				freeVar = &objects.ObjectPtr{Value: &val}
				v.stack[sp] = freeVar
			}

			v.stack[v.sp] = freeVar
			v.sp++

		case compiler.OpSetSelFree:
			v.ip += 2
			freeIndex := int(v.curInsts[v.ip-1])
			numSelectors := int(v.curInsts[v.ip])

			// selectors and RHS value
			selectors := make([]objects.Object, numSelectors)
			for i := 0; i < numSelectors; i++ {
				selectors[i] = v.stack[v.sp-numSelectors+i]
			}
			val := v.stack[v.sp-numSelectors-1]
			v.sp -= numSelectors + 1

			if e := indexAssign(*v.curFrame.freeVars[freeIndex].Value, val, selectors); e != nil {
				v.err = e
				return
			}

		case compiler.OpIteratorInit:
			var iterator objects.Object

			dst := v.stack[v.sp-1]
			v.sp--

			if !dst.CanIterate() {
				v.err = fmt.Errorf("not iterable: %s", dst.TypeName())
				return
			}

			iterator = dst.Iterate()
			v.vm.allocs--
			if v.vm.allocs == 0 {
				v.vm.err = ErrObjectAllocLimit
				return
			}

			v.stack[v.sp] = iterator
			v.sp++

		case compiler.OpIteratorNext:
			iterator := v.stack[v.sp-1]
			v.sp--

			hasMore := iterator.(objects.Iterator).Next()

			if hasMore {
				v.stack[v.sp] = objects.TrueValue
			} else {
				v.stack[v.sp] = objects.FalseValue
			}
			v.sp++

		case compiler.OpIteratorKey:
			iterator := v.stack[v.sp-1]
			v.sp--

			val := iterator.(objects.Iterator).Key()

			v.stack[v.sp] = val
			v.sp++

		case compiler.OpIteratorValue:
			iterator := v.stack[v.sp-1]
			v.sp--

			val := iterator.(objects.Iterator).Value()

			v.stack[v.sp] = val
			v.sp++

		default:
			v.err = fmt.Errorf("unknown opcode: %d", v.curInsts[v.ip])
			return
		}
	}
}
