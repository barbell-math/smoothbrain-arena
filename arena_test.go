package sbarena

import (
	"runtime"
	"testing"
	"unsafe"
	"weak"

	sbtest "github.com/barbell-math/smoothbrain-test"
)

type testStruct struct {
	A int
	B float64
	C string
}

type testStruct2 struct {
	testStruct
	D complex64
}

func TestAllocSimple(t *testing.T) {
	a := NewArena(0)
	one, err := Alloc[testStruct](a)
	sbtest.Nil(t, err)
	*one.Value() = testStruct{A: 1, B: 1, C: "one"}
	sbtest.Eq(t, *one.Value(), testStruct{A: 1, B: 1, C: "one"})

	vals := [6]weak.Pointer[testStruct]{}
	for i, str := range []string{"one", "two", "three", "four", "five", "six"} {
		iterV, err := Alloc[testStruct](a)
		sbtest.Nil(t, err)
		*iterV.Value() = testStruct{A: i, B: float64(i), C: str}
		vals[i] = iterV
	}

	for i, str := range []string{"one", "two", "three", "four", "five", "six"} {
		sbtest.Eq(t, *vals[i].Value(), testStruct{A: i, B: float64(i), C: str})
	}
}

func TestAllocMultipleBuckets(t *testing.T) {
	a := NewArena(unsafe.Sizeof(testStruct{}) * 3)

	vals := [6]weak.Pointer[testStruct]{}
	for i, str := range []string{"one", "two", "three", "four", "five", "six"} {
		iterV, err := Alloc[testStruct](a)
		sbtest.Nil(t, err)
		*iterV.Value() = testStruct{A: i, B: float64(i), C: str}
		vals[i] = iterV
	}
	sbtest.Eq(t, 2, NumBuckets(a))
	sbtest.Eq(t, uintptr(unsafe.Sizeof(testStruct{})*6), TotalMemBytes(a))

	for i, str := range []string{"one", "two", "three", "four", "five", "six"} {
		sbtest.Eq(t, *vals[i].Value(), testStruct{A: i, B: float64(i), C: str})
	}
}

func TestAllocMultipleBucketsUnevenWrap(t *testing.T) {
	a := NewArena(unsafe.Sizeof(testStruct{})*3 - 1)

	vals := [6]weak.Pointer[testStruct]{}
	for i, str := range []string{"one", "two", "three", "four", "five", "six"} {
		iterV, err := Alloc[testStruct](a)
		sbtest.Nil(t, err)
		*iterV.Value() = testStruct{A: i, B: float64(i), C: str}
		vals[i] = iterV
	}
	sbtest.Eq(t, 3, NumBuckets(a))
	sbtest.Eq(t, uintptr(unsafe.Sizeof(testStruct{})*3-1)*3, TotalMemBytes(a))

	for i, str := range []string{"one", "two", "three", "four", "five", "six"} {
		sbtest.Eq(t, *vals[i].Value(), testStruct{A: i, B: float64(i), C: str})
	}
}

func TestAllocMultipleBucketsValueToLarge(t *testing.T) {
	a := NewArena(unsafe.Sizeof(testStruct{}))
	one, err := Alloc[testStruct2](a)
	sbtest.ContainsError(t, ValueToLargeErr, err)
	sbtest.Nil(t, one.Value())
}

func TestReset(t *testing.T) {
	a := NewArena(unsafe.Sizeof(testStruct{}) * 3)

	vals := [6]weak.Pointer[testStruct]{}
	for i, str := range []string{"one", "two", "three", "four", "five", "six"} {
		iterV, err := Alloc[testStruct](a)
		sbtest.Nil(t, err)
		*iterV.Value() = testStruct{A: i, B: float64(i), C: str}
		vals[i] = iterV
	}
	sbtest.Eq(t, 2, NumBuckets(a))
	sbtest.Eq(t, uintptr(unsafe.Sizeof(testStruct{})*6), TotalMemBytes(a))

	Reset(a)

	// Test that pntrs are still useable, though likely invalid
	for i, str := range []string{"one", "two", "three", "four", "five", "six"} {
		sbtest.Eq(t, *vals[i].Value(), testStruct{A: i, B: float64(i), C: str})
	}

	vals2 := [6]weak.Pointer[testStruct]{}
	for i, str := range []string{"six", "five", "four", "three", "two", "one"} {
		iterV, err := Alloc[testStruct](a)
		sbtest.Nil(t, err)
		*iterV.Value() = testStruct{A: i, B: float64(i), C: str}
		vals2[i] = iterV
	}
	sbtest.Eq(t, 2, NumBuckets(a))
	sbtest.Eq(t, uintptr(unsafe.Sizeof(testStruct{})*6), TotalMemBytes(a))

	// Test that pntrs are still useable, though note that the data they point
	// to has now changed.
	for i, str := range []string{"six", "five", "four", "three", "two", "one"} {
		sbtest.Eq(t, *vals[i].Value(), testStruct{A: i, B: float64(i), C: str})
	}
	for i, str := range []string{"six", "five", "four", "three", "two", "one"} {
		sbtest.Eq(t, *vals2[i].Value(), testStruct{A: i, B: float64(i), C: str})
	}
}

func TestClear(t *testing.T) {
	a := NewArena(unsafe.Sizeof(testStruct{}) * 3)

	vals := [6]weak.Pointer[testStruct]{}
	for i, str := range []string{"one", "two", "three", "four", "five", "six"} {
		iterV, err := Alloc[testStruct](a)
		sbtest.Nil(t, err)
		*iterV.Value() = testStruct{A: i, B: float64(i), C: str}
		vals[i] = iterV
	}
	sbtest.Eq(t, 2, NumBuckets(a))
	sbtest.Eq(t, uintptr(unsafe.Sizeof(testStruct{})*6), TotalMemBytes(a))

	Clear(a)
	// Has to be called to force the GC to collect the old arena vals
	runtime.GC()

	// Test that pntrs are no longer useable
	for i := range []string{"one", "two", "three", "four", "five", "six"} {
		sbtest.Nil(t, vals[i].Value())
	}

	vals2 := [6]weak.Pointer[testStruct]{}
	for i, str := range []string{"six", "five", "four", "three", "two", "one"} {
		iterV, err := Alloc[testStruct](a)
		sbtest.Nil(t, err)
		*iterV.Value() = testStruct{A: i, B: float64(i), C: str}
		vals2[i] = iterV
	}
	sbtest.Eq(t, 2, NumBuckets(a))
	sbtest.Eq(t, uintptr(unsafe.Sizeof(testStruct{})*6), TotalMemBytes(a))

	// Test that pntrs are no longer useable
	for i := range []string{"six", "five", "four", "three", "two", "one"} {
		sbtest.Nil(t, vals[i].Value())
	}
	for i, str := range []string{"six", "five", "four", "three", "two", "one"} {
		sbtest.Eq(t, *vals2[i].Value(), testStruct{A: i, B: float64(i), C: str})
	}
}
