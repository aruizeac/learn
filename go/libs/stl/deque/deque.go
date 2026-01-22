package deque

import "iter"

// A Deque is a double-ended queue. It supports O(1) push and pop operations on both ends.
type Deque[T any] struct {
	items []T
}

// NewDeque returns a new Deque.
func NewDeque[T any](opts ...Option) *Deque[T] {
	return NewDequeFromSlice[T](nil, opts...)
}

// NewDequeFromSlice returns a new Deque initialized with the given slice.
func NewDequeFromSlice[T any](items []T, opts ...Option) *Deque[T] {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	if items == nil {
		items = make([]T, 0, o.capacity)
	}
	return &Deque[T]{
		items: items,
	}
}

type options struct {
	capacity int
}

// Option is a function that configures a Deque.
type Option func(*options)

// WithCapacity sets the initial capacity of the deque.
func WithCapacity(capacity int) Option {
	return func(o *options) { o.capacity = capacity }
}

// PushBack pushes an item onto the back of the deque.
func (d *Deque[T]) PushBack(x T) {
	d.items = append(d.items, x)
}

// PopFront pops an item off the front of the deque.
func (d *Deque[T]) PopFront() T {
	x := d.items[0]
	d.items = d.items[1:]
	return x
}

// PopBack pops an item off the back of the deque.
func (d *Deque[T]) PopBack() T {
	x := d.items[len(d.items)-1]
	d.items = d.items[:len(d.items)-1]
	return x
}

// Front peeks the front item of the deque.
func (d *Deque[T]) Front() T { return d.items[0] }

// Back peeks the back item of the deque.
func (d *Deque[T]) Back() T { return d.items[len(d.items)-1] }

// Len returns the number of items in the deque.
func (d *Deque[T]) Len() int { return len(d.items) }

// Empty returns true if the deque is empty.
func (d *Deque[T]) Empty() bool { return len(d.items) == 0 }

// Clear removes all items from the deque.
func (d *Deque[T]) Clear() { d.items = d.items[:0] }

// Values returns a sequence of all items in the deque, in order.
func (d *Deque[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := range d.items {
			if !yield(d.items[i]) {
				return
			}
		}
	}
}
