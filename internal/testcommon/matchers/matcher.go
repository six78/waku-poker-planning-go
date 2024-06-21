package matchers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type Matcher struct {
	t         *testing.T
	triggered chan interface{}
}

func NewMatcher(t *testing.T) *Matcher {
	return &Matcher{
		t:         t,
		triggered: make(chan interface{}, 42),
	}
}

func (m *Matcher) Wait() interface{} {
	select {
	case <-time.After(1 * time.Second):
		require.Fail(m.t, "timeout waiting for matched call")
	case result := <-m.triggered:
		return result
	}
	return nil
}
