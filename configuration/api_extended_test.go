// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

// +build !node

package configuration

import (
	"math/big"
	"testing"
	"time"

	"github.com/insolar/insconfig"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/require"
)

type testPathGetter struct {
	Path string
}

func (g testPathGetter) GetConfigPath() string {
	return g.Path
}

func TestConfigLoad(t *testing.T) {
	cfg := APIExtended{}
	params := insconfig.Params{
		EnvPrefix:        "observerapi",
		ViperHooks:       []mapstructure.DecodeHookFunc{ToBigIntHookFunc()},
		ConfigPathGetter: testPathGetter{"./testdata/observerapi.yaml"},
	}
	insConfigurator := insconfig.New(params)
	if err := insConfigurator.Load(&cfg); err != nil {
		panic(err)
	}

	require.Equal(t, big.NewInt(2000000000), cfg.FeeAmount)
	require.Equal(t, time.Second*3, cfg.DB.AttemptInterval)
}
