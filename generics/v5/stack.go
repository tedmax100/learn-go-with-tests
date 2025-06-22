package v5

type StackOfInts = Stack
type StackOfStrings = Stack

type Stack struct {
	values []any
}

func (s *Stack) Push(value any) {
	s.values = append(s.values, value)
}

func (s *Stack) IsEmpty() bool {
	return len(s.values) == 0
}

func (s *Stack) Pop() (any, bool) {
	if s.IsEmpty() {
		var zero any
		return zero, false
	}

	index := len(s.values) - 1
	el := s.values[index]
	s.values = s.values[:index]
	return el, true
}
