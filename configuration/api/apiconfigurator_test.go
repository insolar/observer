// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package api

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigLoad(t *testing.T) {
	actual := load("./testdata")
	require.Equal(t, big.NewInt(2000000000), actual.FeeAmount)
	require.Equal(t, time.Second*3, actual.DB.AttemptInterval)
}
