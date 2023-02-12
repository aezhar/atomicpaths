package atomicpaths

type state int

func (s *state) is(f state) bool {
	return *s&f == f
}

func (s *state) set(f state) {
	*s |= f
}

const (
	closed state = 1 << iota
	placed
	synced
)
