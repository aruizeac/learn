package textedit

import (
	"slices"
	"stl/deque"
)

type Document struct {
	Content []byte
}

// Action

type Action interface {
	Execute(document *Document) error
	Reverse(document *Document) error
}

// Editor

type Editor struct {
	content *Document

	undo *deque.Deque[Action]
	redo *deque.Deque[Action]
}

// NewEditor returns a new text Editor.
func NewEditor(doc *Document) *Editor {
	return &Editor{
		content: doc,
		undo:    deque.NewDeque[Action](deque.WithCapacity(32)), // up to 32 actions
		redo:    deque.NewDeque[Action](deque.WithCapacity(32)),
	}
}

func (e *Editor) Execute(a Action) error {
	if err := a.Execute(e.content); err != nil {
		return err
	}
	e.undo.PushBack(a)
	e.redo.Clear()
	return nil
}

func (e *Editor) Undo() error {
	if e.undo.Empty() {
		return nil
	}
	a := e.undo.PopBack()
	if err := a.Reverse(e.content); err != nil {
		return err
	}
	e.redo.PushBack(a)
	return nil
}

func (e *Editor) Redo() error {
	if e.redo.Empty() {
		return nil
	}
	a := e.redo.PopBack()
	if err := a.Execute(e.content); err != nil {
		return err
	}
	e.undo.PushBack(a)
	return nil
}

// Action Implementations

// InsertAction is an action that inserts a string at a given position in a file.
type InsertAction struct {
	pos int
	s   string
}

var _ Action = (*InsertAction)(nil)

func NewInsertAction(pos int, s string) *InsertAction {
	return &InsertAction{pos, s}
}

func (i InsertAction) Execute(document *Document) error {
	document.Content = slices.Insert(document.Content, i.pos, []byte(i.s)...)
	return nil
}

func (i InsertAction) Reverse(document *Document) error {
	document.Content = slices.Delete(document.Content, i.pos, i.pos+len(i.s))
	return nil
}

// DeleteAction is an action that deletes a range of bytes from a file.
type DeleteAction struct {
	pos     int
	end     int
	removed []byte
}

var _ Action = (*DeleteAction)(nil)

// NewDeleteAction returns a new DeleteAction.
func NewDeleteAction(pos, end int) *DeleteAction {
	return &DeleteAction{pos, end, nil}
}

func (d *DeleteAction) Execute(document *Document) error {
	d.removed = make([]byte, d.end-d.pos)
	copy(d.removed, document.Content[d.pos:d.end])
	document.Content = slices.Delete(document.Content, d.pos, d.end)
	return nil
}

func (d *DeleteAction) Reverse(document *Document) error {
	if d.removed == nil {
		return nil
	}
	document.Content = slices.Insert(document.Content, d.pos, d.removed...)
	return nil
}
