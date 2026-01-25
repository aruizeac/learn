package lint

import (
	"bufio"
	"errors"
	"io"
	"stl/deque"
	"sync"
)

type Error struct {
	Message string
	Line    int
	Pos     int
}

var _ error = (*Error)(nil)

func (e Error) Error() string {
	return e.Message
}

var _openerMap = sync.OnceValue(func() map[rune]rune {
	return map[rune]rune{
		'(': ')',
		'[': ']',
		'{': '}',
	}
})()

// Lint checks the given content for mismatched parentheses, brackets, and braces.
func Lint(content io.Reader) error {
	scanner := bufio.NewScanner(content)
	openers := deque.NewDeque[opener]()
	lineNum := 0
	errs := make([]error, 0)
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		for i, char := range line {
			if char == '(' || char == '[' || char == '{' {
				openers.PushBack(opener{char: char, line: lineNum, pos: i})
				continue
			}

			isCloser := char == ')' || char == ']' || char == '}'
			if isCloser && openers.Empty() {
				errs = append(errs, &Error{Message: "unexpected closer", Line: lineNum, Pos: i})
				continue
			} else if !isCloser {
				continue
			}

			lastOpener := openers.Back()
			if _openerMap[lastOpener.char] != char {
				errs = append(errs, &Error{Message: "mismatched opener and closer", Line: lastOpener.line, Pos: lastOpener.pos})
			}
			openers.PopBack() // pop anyway
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	if openers.Empty() {
		return errors.Join(errs...)
	}

	for o := range openers.Values() {
		errs = append(errs, &Error{Message: "unmatched opener", Line: o.line, Pos: o.pos})
	}
	return errors.Join(errs...)
}

type opener struct {
	char rune
	line int
	pos  int
}
