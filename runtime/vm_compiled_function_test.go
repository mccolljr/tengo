package runtime

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/d5/tengo/objects"
)

func TestCompiledFunctionType(t *testing.T) {
	const (
		realName = "objects.CompiledFunction"
		fakeName = "compiledFunction"
	)

	real := objects.CompiledFunction{}
	fake := compiledFunction{}

	realSize := unsafe.Sizeof(real)
	fakeSize := unsafe.Sizeof(fake)
	if realSize != fakeSize {
		t.Fatalf("size mismatch: %d bytes (%s) / %d bytes (%s)",
			realSize, realName, fakeSize, fakeName)
	}

	realType := reflect.TypeOf(real)
	fakeType := reflect.TypeOf(fake)

	realKind := realType.Kind()
	fakeKind := fakeType.Kind()
	if realKind != fakeKind {
		t.Fatalf("kind mismatch: %s (%s) / %s (%s)",
			realKind, realName, fakeKind, fakeName)
	}

	realFields := realType.NumField()
	fakeFields := fakeType.NumField()
	if realFields != fakeFields {
		t.Fatalf("field count mismatch: %d (%s) / %d (%s)",
			realFields, realName, fakeFields, fakeName)
	}

	for i := 0; i < realFields; i++ {
		realField := realType.Field(i)
		fakeField := fakeType.Field(i)
		if realField.Type != fakeField.Type {
			t.Fatalf("field type mismatch: %s %s (%s) / %s %s (%s)",
				realField.Name, realField.Type, realName,
				fakeField.Name, fakeField.Type, fakeName)
		}
	}
}
