package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_makeBeautyMetrics(t *testing.T) {
	obs := Make(context.Background())
	metrics := MakeBeautyMetrics(obs, "processed")
	require.NotNil(t, metrics)
}
