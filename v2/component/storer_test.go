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

package component

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/v2/configuration"
	"github.com/insolar/observer/v2/internal/app/observer"
)

func Test_makeStorer(t *testing.T) {
	cfg := configuration.Default()
	obs := makeObservability()
	conn := makeConnectivity(cfg, obs)
	storer := makeStorer(obs, conn)

	b := &beauty{
		transfers: []*observer.DepositTransfer{{}},
	}
	require.NotPanics(t, func() {
		storer(b)
	})
}
