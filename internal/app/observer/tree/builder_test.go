package tree

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuilder_Build(t *testing.T) {
	b := NewBuilder(nil)
	require.NotNil(t, b)
}
