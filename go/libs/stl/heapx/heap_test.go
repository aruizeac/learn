package heapx

import (
	"container/heap"
	"slices"
	"testing"
)

func TestNewHeap(t *testing.T) {
	tests := []struct {
		name         string
		lessFunc     LessFunc[int]
		opts         []Option
		wantLen      int
		wantCapacity int
	}{
		{
			name:         "min heap without options",
			lessFunc:     func(a, b int) bool { return a < b },
			opts:         nil,
			wantLen:      0,
			wantCapacity: 0,
		},
		{
			name:         "min heap with size option",
			lessFunc:     func(a, b int) bool { return a < b },
			opts:         []Option{WithHeapSize(10)},
			wantLen:      0,
			wantCapacity: 10,
		},
		{
			name:         "max heap without options",
			lessFunc:     func(a, b int) bool { return a > b },
			opts:         nil,
			wantLen:      0,
			wantCapacity: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHeap(tt.lessFunc, tt.opts...)

			if h.Len() != tt.wantLen {
				t.Errorf("Len() = %d, want %d", h.Len(), tt.wantLen)
			}
			if cap(h.items) != tt.wantCapacity {
				t.Errorf("cap(items) = %d, want %d", cap(h.items), tt.wantCapacity)
			}
		})
	}
}

func TestHeap_Len(t *testing.T) {
	tests := []struct {
		name    string
		items   []int
		wantLen int
	}{
		{
			name:    "empty heap",
			items:   []int{},
			wantLen: 0,
		},
		{
			name:    "single item",
			items:   []int{1},
			wantLen: 1,
		},
		{
			name:    "multiple items",
			items:   []int{1, 2, 3, 4, 5},
			wantLen: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHeap(func(a, b int) bool { return a < b })
			for _, item := range tt.items {
				heap.Push(h, item)
			}

			if h.Len() != tt.wantLen {
				t.Errorf("Len() = %d, want %d", h.Len(), tt.wantLen)
			}
		})
	}
}

func TestHeap_MinHeap(t *testing.T) {
	tests := []struct {
		name      string
		pushItems []int
		wantOrder []int
	}{
		{
			name:      "ascending order input",
			pushItems: []int{1, 2, 3, 4, 5},
			wantOrder: []int{1, 2, 3, 4, 5},
		},
		{
			name:      "descending order input",
			pushItems: []int{5, 4, 3, 2, 1},
			wantOrder: []int{1, 2, 3, 4, 5},
		},
		{
			name:      "random order input",
			pushItems: []int{3, 1, 4, 1, 5, 9, 2, 6},
			wantOrder: []int{1, 1, 2, 3, 4, 5, 6, 9},
		},
		{
			name:      "single item",
			pushItems: []int{42},
			wantOrder: []int{42},
		},
		{
			name:      "duplicates",
			pushItems: []int{5, 5, 5, 5},
			wantOrder: []int{5, 5, 5, 5},
		},
		{
			name:      "negative numbers",
			pushItems: []int{-3, -1, -4, -1, -5},
			wantOrder: []int{-5, -4, -3, -1, -1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHeap(func(a, b int) bool { return a < b })
			for _, item := range tt.pushItems {
				heap.Push(h, item)
			}

			var got []int
			for h.Len() > 0 {
				got = append(got, heap.Pop(h).(int))
			}

			if !slices.Equal(got, tt.wantOrder) {
				t.Errorf("pop order = %v, want %v", got, tt.wantOrder)
			}
		})
	}
}

func TestHeap_MaxHeap(t *testing.T) {
	tests := []struct {
		name      string
		pushItems []int
		wantOrder []int
	}{
		{
			name:      "ascending order input",
			pushItems: []int{1, 2, 3, 4, 5},
			wantOrder: []int{5, 4, 3, 2, 1},
		},
		{
			name:      "descending order input",
			pushItems: []int{5, 4, 3, 2, 1},
			wantOrder: []int{5, 4, 3, 2, 1},
		},
		{
			name:      "random order input",
			pushItems: []int{3, 1, 4, 1, 5, 9, 2, 6},
			wantOrder: []int{9, 6, 5, 4, 3, 2, 1, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHeap(func(a, b int) bool { return a > b }) // max heap
			for _, item := range tt.pushItems {
				heap.Push(h, item)
			}

			var got []int
			for h.Len() > 0 {
				got = append(got, heap.Pop(h).(int))
			}

			if !slices.Equal(got, tt.wantOrder) {
				t.Errorf("pop order = %v, want %v", got, tt.wantOrder)
			}
		})
	}
}

func TestHeap_Peek(t *testing.T) {
	tests := []struct {
		name      string
		lessFunc  LessFunc[int]
		pushItems []int
		wantPeek  int
	}{
		{
			name:      "min heap peek",
			lessFunc:  func(a, b int) bool { return a < b },
			pushItems: []int{5, 3, 7, 1, 9},
			wantPeek:  1,
		},
		{
			name:      "max heap peek",
			lessFunc:  func(a, b int) bool { return a > b },
			pushItems: []int{5, 3, 7, 1, 9},
			wantPeek:  9,
		},
		{
			name:      "single item",
			lessFunc:  func(a, b int) bool { return a < b },
			pushItems: []int{42},
			wantPeek:  42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHeap(tt.lessFunc)
			for _, item := range tt.pushItems {
				heap.Push(h, item)
			}

			got := h.Peek()
			if got != tt.wantPeek {
				t.Errorf("Peek() = %d, want %d", got, tt.wantPeek)
			}

			// Peek should not modify the heap
			if h.Len() != len(tt.pushItems) {
				t.Errorf("Peek() modified heap length: got %d, want %d", h.Len(), len(tt.pushItems))
			}
		})
	}
}

func TestHeap_PushItemPopItem(t *testing.T) {
	tests := []struct {
		name      string
		pushItems []int
		popCount  int
		wantPops  []int
		wantLen   int
	}{
		{
			name:      "push and pop all",
			pushItems: []int{3, 1, 2},
			popCount:  3,
			wantPops:  []int{1, 2, 3},
			wantLen:   0,
		},
		{
			name:      "push and pop partial",
			pushItems: []int{5, 3, 7, 1},
			popCount:  2,
			wantPops:  []int{1, 3},
			wantLen:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHeap(func(a, b int) bool { return a < b })
			for _, item := range tt.pushItems {
				h.PushItem(item)
			}
			heap.Init(h) // need to heapify after using PushItem directly

			var got []int
			for i := 0; i < tt.popCount; i++ {
				got = append(got, heap.Pop(h).(int))
			}

			if !slices.Equal(got, tt.wantPops) {
				t.Errorf("popped items = %v, want %v", got, tt.wantPops)
			}
			if h.Len() != tt.wantLen {
				t.Errorf("remaining Len() = %d, want %d", h.Len(), tt.wantLen)
			}
		})
	}
}

func TestHeap_Values(t *testing.T) {
	tests := []struct {
		name      string
		lessFunc  LessFunc[int]
		pushItems []int
		wantOrder []int
	}{
		{
			name:      "min heap values",
			lessFunc:  func(a, b int) bool { return a < b },
			pushItems: []int{5, 3, 7, 1, 9},
			wantOrder: []int{1, 3, 5, 7, 9},
		},
		{
			name:      "max heap values",
			lessFunc:  func(a, b int) bool { return a > b },
			pushItems: []int{5, 3, 7, 1, 9},
			wantOrder: []int{9, 7, 5, 3, 1},
		},
		{
			name:      "empty heap",
			lessFunc:  func(a, b int) bool { return a < b },
			pushItems: []int{},
			wantOrder: []int{},
		},
		{
			name:      "single item",
			lessFunc:  func(a, b int) bool { return a < b },
			pushItems: []int{42},
			wantOrder: []int{42},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHeap(tt.lessFunc)
			for _, item := range tt.pushItems {
				heap.Push(h, item)
			}

			var got []int
			for item := range h.Values() {
				got = append(got, item)
			}

			if len(got) == 0 && len(tt.wantOrder) == 0 {
				return // both empty, pass
			}
			if !slices.Equal(got, tt.wantOrder) {
				t.Errorf("Values() = %v, want %v", got, tt.wantOrder)
			}

			// Values() should empty the heap
			if h.Len() != 0 {
				t.Errorf("heap should be empty after Values(), got Len() = %d", h.Len())
			}
		})
	}
}

func TestHeap_ValuesEarlyBreak(t *testing.T) {
	h := NewHeap(func(a, b int) bool { return a < b })
	for _, item := range []int{5, 3, 7, 1, 9} {
		heap.Push(h, item)
	}

	var got []int
	count := 0
	for item := range h.Values() {
		got = append(got, item)
		count++
		if count >= 2 {
			break
		}
	}

	wantFirst := []int{1, 3}
	if !slices.Equal(got, wantFirst) {
		t.Errorf("first 2 values = %v, want %v", got, wantFirst)
	}
}

func TestHeap_Swap(t *testing.T) {
	tests := []struct {
		name  string
		items []int
		i, j  int
		wantI int
		wantJ int
	}{
		{
			name:  "swap first and last",
			items: []int{1, 2, 3},
			i:     0,
			j:     2,
			wantI: 3,
			wantJ: 1,
		},
		{
			name:  "swap adjacent",
			items: []int{1, 2, 3},
			i:     0,
			j:     1,
			wantI: 2,
			wantJ: 1,
		},
		{
			name:  "swap same index",
			items: []int{1, 2, 3},
			i:     1,
			j:     1,
			wantI: 2,
			wantJ: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Heap[int]{
				items:    slices.Clone(tt.items),
				lessFunc: func(a, b int) bool { return a < b },
			}

			h.Swap(tt.i, tt.j)

			if h.items[tt.i] != tt.wantI {
				t.Errorf("items[%d] = %d, want %d", tt.i, h.items[tt.i], tt.wantI)
			}
			if h.items[tt.j] != tt.wantJ {
				t.Errorf("items[%d] = %d, want %d", tt.j, h.items[tt.j], tt.wantJ)
			}
		})
	}
}

func TestHeap_Less(t *testing.T) {
	tests := []struct {
		name     string
		lessFunc LessFunc[int]
		items    []int
		i, j     int
		want     bool
	}{
		{
			name:     "min heap less true",
			lessFunc: func(a, b int) bool { return a < b },
			items:    []int{1, 5, 3},
			i:        0,
			j:        1,
			want:     true,
		},
		{
			name:     "min heap less false",
			lessFunc: func(a, b int) bool { return a < b },
			items:    []int{5, 1, 3},
			i:        0,
			j:        1,
			want:     false,
		},
		{
			name:     "max heap less true",
			lessFunc: func(a, b int) bool { return a > b },
			items:    []int{5, 1, 3},
			i:        0,
			j:        1,
			want:     true,
		},
		{
			name:     "equal items",
			lessFunc: func(a, b int) bool { return a < b },
			items:    []int{3, 3, 3},
			i:        0,
			j:        1,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Heap[int]{
				items:    tt.items,
				lessFunc: tt.lessFunc,
			}

			got := h.Less(tt.i, tt.j)
			if got != tt.want {
				t.Errorf("Less(%d, %d) = %v, want %v", tt.i, tt.j, got, tt.want)
			}
		})
	}
}

func TestHeap_CustomType(t *testing.T) {
	type item struct {
		priority int
		value    string
	}

	tests := []struct {
		name      string
		lessFunc  LessFunc[item]
		pushItems []item
		wantOrder []string
	}{
		{
			name:     "priority queue min",
			lessFunc: func(a, b item) bool { return a.priority < b.priority },
			pushItems: []item{
				{priority: 3, value: "low"},
				{priority: 1, value: "high"},
				{priority: 2, value: "medium"},
			},
			wantOrder: []string{"high", "medium", "low"},
		},
		{
			name:     "priority queue max",
			lessFunc: func(a, b item) bool { return a.priority > b.priority },
			pushItems: []item{
				{priority: 3, value: "high"},
				{priority: 1, value: "low"},
				{priority: 2, value: "medium"},
			},
			wantOrder: []string{"high", "medium", "low"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHeap(tt.lessFunc)
			for _, it := range tt.pushItems {
				heap.Push(h, it)
			}

			var got []string
			for h.Len() > 0 {
				got = append(got, heap.Pop(h).(item).value)
			}

			if !slices.Equal(got, tt.wantOrder) {
				t.Errorf("pop order = %v, want %v", got, tt.wantOrder)
			}
		})
	}
}

func TestHeap_WithHeapSize(t *testing.T) {
	tests := []struct {
		name         string
		size         int
		wantCapacity int
	}{
		{
			name:         "size 0",
			size:         0,
			wantCapacity: 0,
		},
		{
			name:         "size 10",
			size:         10,
			wantCapacity: 10,
		},
		{
			name:         "size 100",
			size:         100,
			wantCapacity: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHeap(func(a, b int) bool { return a < b }, WithHeapSize(tt.size))

			if cap(h.items) != tt.wantCapacity {
				t.Errorf("cap(items) = %d, want %d", cap(h.items), tt.wantCapacity)
			}
		})
	}
}

func TestHeap_HeapInterface(t *testing.T) {
	// Verify that Heap implements heap.Interface
	var _ heap.Interface = (*Heap[int])(nil)
	var _ heap.Interface = (*Heap[string])(nil)

	type custom struct{ v int }
	var _ heap.Interface = (*Heap[custom])(nil)
}
