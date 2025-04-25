package sbarena

import (
	"errors"
	"sync/atomic"
	"unsafe"
	"weak"

	sberr "github.com/barbell-math/smoothbrain-errs"
)

type (
	bucket []byte

	// A dynamic arena allocator that is backed by buckets. Objects that are
	// larger than the bucket size cannot be stored in the area. The bucket size
	// can be specified when calling [NewArena].
	//
	// Go is a GC'ed language, so you cannot control exactly when the GC will
	// free the arena but when it does all of the objects that it stores will
	// be freed along with it. The GC cleaning up the Arena struct is equivalent
	// to freeing all of the memory.
	//
	// An Arena is thread safe for allocations and frees, though once the arena
	// is freed all pointers to the data it contained will be invalidated and
	// set to nil.
	//
	// An Arena should *not* be copied by value, this will invalidate the
	// atomics protecting allocation operations.
	Arena struct {
		buckets    []bucket
		curBucket  int
		bytesLeft  uintptr
		bucketSize uintptr
		writing    atomic.Bool
	}
)

const (
	// 64 Kib. The default bucket size used when a bucket size <=0 is supplied
	// to [NewArena].
	DefaultBlockSize uintptr = 65536
)

var (
	ValueToLargeErr = errors.New(
		"The supplied value was to large to place in the arena",
	)
)

func newBucket(size uintptr) bucket {
	return make(bucket, size, size)
}

// Creates a new [Arena] allocator, initilizing it to use `bucketSizeBytes`
// bucket size.
func NewArena(bucketSizeBytes uintptr) *Arena {
	if bucketSizeBytes <= 0 {
		bucketSizeBytes = DefaultBlockSize
	}

	return &Arena{
		buckets:    []bucket{newBucket(uintptr(bucketSizeBytes))},
		curBucket:  0,
		bytesLeft:  uintptr(bucketSizeBytes),
		bucketSize: uintptr(bucketSizeBytes),
	}
}

// Returns the bucket size for the given arena.
func BucketSizeBytes(a *Arena) uintptr {
	return a.bucketSize
}

// Gets the number of buckets that the arena has currently allocated.
func NumBuckets(a *Arena) int {
	return len(a.buckets)
}

// Returns the total number of bytes the arena has allocated accross all
// buckets.
func TotalMemBytes(a *Arena) uintptr {
	return a.bucketSize * uintptr(len(a.buckets))
}

// Allocates enough space in the arena to hold a value of type T. The size of T
// must be less than the bucket size the allocator was initilized with,
// otherwise a [ValueToLargeErr] will be returned.
func Alloc[T any](a *Arena) (weak.Pointer[T], error) {
	var tmp T
	size := unsafe.Sizeof(tmp)
	if size > a.bucketSize {
		return weak.Make[T](nil), sberr.Wrap(
			ValueToLargeErr,
			"Requested size: %d Got Size: %d",
			size, a.bucketSize,
		)
	}

	for !a.writing.CompareAndSwap(false, true) {
	}

	if len(a.buckets) == 0 {
		a.buckets = append(a.buckets, newBucket(a.bucketSize))
		a.bytesLeft = a.bucketSize
		a.curBucket = 0
	} else if a.bytesLeft < size {
		if a.curBucket == len(a.buckets)-1 {
			a.buckets = append(a.buckets, newBucket(a.bucketSize))
		}
		a.curBucket++
		a.bytesLeft = a.bucketSize
	}

	ptr := unsafe.Pointer(&a.buckets[a.curBucket][a.bucketSize-a.bytesLeft])
	a.bytesLeft -= size
	a.writing.Store(false)

	return weak.Make((*T)(ptr)), nil
}

// Resets the internal state of the arena so that it starts to reuse memory,
// overwriting the memory it previously used.
//
// No new memory will be allocated and as such all other pointers that reference
// this arenas memory can still be used, though they are no longer guarinteed to
// point to valid values.
func Reset(a *Arena) {
	for !a.writing.CompareAndSwap(false, true) {
	}

	a.bytesLeft = a.bucketSize
	a.curBucket = 0

	a.writing.Store(false)
}

// Frees all of the memory that the arena allocated. Calling this function will
// result in all other pointers that reference this arenas memory to be set to
// nil.
//
// The arena can still be used after this operation, it will allocate more
// memory as needed. If this arena is used to allocate more memory the old
// memory will not be reused.
func Clear(a *Arena) {
	for !a.writing.CompareAndSwap(false, true) {
	}

	a.buckets = []bucket{}
	a.bytesLeft = a.bucketSize
	a.curBucket = 0

	a.writing.Store(false)
}
