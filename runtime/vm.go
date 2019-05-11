package runtime

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/d5/tengo/compiler"
	"github.com/d5/tengo/compiler/source"
	"github.com/d5/tengo/objects"
)

const (
	// StackSize is the maximum stack size.
	StackSize = 2048

	// GlobalsSize is the maximum number of global variables.
	GlobalsSize = 1024

	// MaxFrames is the maximum number of function frames.
	MaxFrames = 1024

	// MaxThreads is the maximum number of threads
	MaxThreads = 8

	// MainThreadID is the ID of the main thread (always 1)
	MainThreadID = 1
)

// VM is a virtual machine that executes the bytecode compiled by Compiler.
type VM struct {
	constants []objects.Object
	globals   []objects.Object
	fileSet   *source.FileSet
	mainFn    *objects.CompiledFunction

	threads       map[uint64]*Thread
	threadResults map[uint64]objects.Object
	threadPool    chan *Thread
	threadCount   uint64
	currentThread *Thread

	aborting  int64
	maxAllocs int64
	allocs    int64
	err       error

	gil sync.Mutex
}

func (v *VM) acquire(threadID uint64) {
	v.gil.Lock()
	t := v.threads[threadID]
	if t == nil {
		panic(fmt.Errorf("unknown thread %d", threadID))
	}
	v.currentThread = t
}

func (v *VM) release() {
	v.gil.Unlock()
}

// NewVM creates a VM.
func NewVM(bytecode *compiler.Bytecode, globals []objects.Object, maxAllocs int64) *VM {
	if globals == nil {
		globals = make([]objects.Object, GlobalsSize)
	}

	v := &VM{
		constants: bytecode.Constants,
		globals:   globals,
		fileSet:   bytecode.FileSet,
		mainFn:    bytecode.MainFunction,
		maxAllocs: maxAllocs,
	}

	return v
}

func (v *VM) createThread(fn *objects.CompiledFunction) *Thread {
	t := <-v.threadPool

	v.threadCount++
	*t = Thread{
		vm:          v,
		id:          v.threadCount,
		sp:          fn.NumLocals,
		framesIndex: 1,
		ip:          -1,
	}

	if v.currentThread == nil {
		t.isMain = true
	} else {
		t.parentID = v.currentThread.id
	}

	t.frames[0].fn = fn
	t.frames[0].ip = -1
	t.curFrame = &t.frames[0]
	t.curInsts = t.curFrame.fn.Instructions

	v.threads[t.id] = t
	return t
}

func (v *VM) initThreads(maxThreads int) {
	v.threadCount = 0
	v.threads = make(map[uint64]*Thread, maxThreads)
	v.threadResults = make(map[uint64]objects.Object, maxThreads)
	v.threadPool = make(chan *Thread, MaxThreads)
	for i := 0; i < MaxThreads; i++ {
		v.threadPool <- &Thread{}
	}
	v.currentThread = nil
}

// Abort aborts the execution.
func (v *VM) Abort() {
	atomic.StoreInt64(&v.aborting, 1)
}

// Call provides a hook for go code to initiate a call to a tengo function
func (v *VM) Call(fn objects.Object, args ...objects.Object) (retVal objects.Object, retErr error) {
	return v.currentThread.call(fn, args)
}

// Run starts the execution.
func (v *VM) Run() (err error) {
	// reset VM states
	v.allocs = v.maxAllocs + 1
	v.initThreads(MaxThreads)

	v.createThread(v.mainFn).execute()

	atomic.StoreInt64(&v.aborting, 0)

	if v.err == nil { // required: allocs are vm-wide so allocation errors appear on the vm
		v.err = v.threads[MainThreadID].err
	}
	err = v.err
	if err != nil {
		filePos := v.fileSet.Position(v.currentThread.curFrame.fn.SourcePos(v.currentThread.ip - 1))
		err = fmt.Errorf("Runtime Error: %s\n\tat %s", err.Error(), filePos)
		for v.currentThread.framesIndex > 1 {
			v.currentThread.framesIndex--
			v.currentThread.curFrame = &v.currentThread.frames[v.currentThread.framesIndex-1]

			filePos = v.fileSet.Position(v.currentThread.curFrame.fn.SourcePos(v.currentThread.curFrame.ip - 1))
			err = fmt.Errorf("%s\n\tat %s", err.Error(), filePos)
		}
		return err
	}

	return nil
}

// IsStackEmpty tests if the stack is empty or not.
func (v *VM) IsStackEmpty() bool {
	if t, ok := v.threads[MainThreadID]; ok {
		return t.sp == 0
	}
	return true // code hasn't run yet so
}

func indexAssign(dst, src objects.Object, selectors []objects.Object) error {
	numSel := len(selectors)

	for sidx := numSel - 1; sidx > 0; sidx-- {
		next, err := dst.IndexGet(selectors[sidx])
		if err != nil {
			if err == objects.ErrNotIndexable {
				return fmt.Errorf("not indexable: %s", dst.TypeName())
			}

			if err == objects.ErrInvalidIndexType {
				return fmt.Errorf("invalid index type: %s", selectors[sidx].TypeName())
			}

			return err
		}

		dst = next
	}

	if err := dst.IndexSet(selectors[0], src); err != nil {
		if err == objects.ErrNotIndexAssignable {
			return fmt.Errorf("not index-assignable: %s", dst.TypeName())
		}

		if err == objects.ErrInvalidIndexValueType {
			return fmt.Errorf("invaid index value type: %s", src.TypeName())
		}

		return err
	}

	return nil
}
