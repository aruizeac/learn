package heapx

import (
	"container/heap"
	"iter"
)

type LessFunc[T any] func(a, b T) bool

type Heap[T any] struct {
	items    []T
	lessFunc LessFunc[T]
}

var _ heap.Interface = (*Heap[int])(nil)

// NewHeap returns a new heap.
func NewHeap[T any](lessFunc LessFunc[T], opts ...Option) *Heap[T] {
	heapOptions := &options{heapSize: 0}
	for _, opt := range opts {
		opt(heapOptions)
	}

	var items []T
	if heapOptions.heapSize > 0 {
		items = make([]T, 0, heapOptions.heapSize)
	} else {
		items = make([]T, 0)
	}
	return &Heap[T]{items: items, lessFunc: lessFunc}
}

func (h *Heap[T]) Len() int { return len(h.items) }

func (h *Heap[T]) Less(i, j int) bool {
	return h.lessFunc(h.items[i], h.items[j])
}
func (h *Heap[T]) Swap(i, j int) { h.items[i], h.items[j] = h.items[j], h.items[i] }

func (h *Heap[T]) Push(x any) {
	h.items = append(h.items, x.(T))
}

func (h *Heap[T]) Pop() any {
	n := len(h.items)
	x := h.items[n-1]
	h.items = h.items[:n-1]
	return x
}

// PushItem pushes an item onto the heap.
//
// It is a convenience method for Push(item).
func (h *Heap[T]) PushItem(x T) {
	h.items = append(h.items, x)
}

// PopItem pops an item off the heap.
//
// It is a convenience method for Pop().
func (h *Heap[T]) PopItem() T {
	n := len(h.items)
	x := h.items[n-1]
	h.items = h.items[:n-1]
	return x
}

func (h *Heap[T]) Peek() T { return h.items[0] } // assumes non-empty

// IndexOf returns the index of the first item satisfying the predicate, or -1 if none.
func (h *Heap[T]) IndexOf(predicate func(v T) bool) int {
	for i, v := range h.items {
		if predicate(v) {
			return i
		}
	}
	return -1
}

// Values returns a sequence of all items in the heap, in order.
// In addition, it pops all items from the heap, making this method unreversible.
func (h *Heap[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		lenSnapshot := len(h.items)
		for i := 0; i < lenSnapshot; i++ {
			if !yield(heap.Pop(h).(T)) {
				return
			}
		}
	}
}

type options struct {
	heapSize int
}

type Option func(*options)

// WithHeapSize sets the initial size of the heap.
func WithHeapSize(size int) Option { return func(o *options) { o.heapSize = size } }
