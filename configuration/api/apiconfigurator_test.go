//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package api

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
	cfg := Configuration{}
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
