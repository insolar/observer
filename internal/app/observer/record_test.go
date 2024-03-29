package observer

import (
	"testing"

	"github.com/insolar/insolar/insolar/record"
	"github.com/stretchr/testify/require"
)

func TestRecord_Marshal(t *testing.T) {
	original := &record.Material{}
	expectedBytes, err := original.Marshal()
	require.NoError(t, err)

	rec := (*Record)(original)
	actualBytes, err := rec.Marshal()

	require.NoError(t, err)
	require.Equal(t, expectedBytes, actualBytes)
}

func TestRecord_Unmarshal(t *testing.T) {
	original := &record.Material{}
	bytes, err := original.Marshal()
	require.NoError(t, err)

	rec := &Record{}
	err = rec.Unmarshal(bytes)
	actual := (*record.Material)(rec)

	require.NoError(t, err)
	require.Equal(t, original, actual)
}
