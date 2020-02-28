// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

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
