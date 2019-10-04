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
	"time"

	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/connectivity"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/observability"
)

func Test_makeStorer(t *testing.T) {
	cfg := configuration.Default()
	obs := observability.Make()
	conn := connectivity.Make(cfg, obs)
	storer := makeStorer(cfg, obs, conn)

	b := &beauty{
		transfers: []*observer.ExtendedTransfer{{}},
	}
	s := &state{}

	cfg.DB.Attempts = 1
	cfg.DB.AttemptInterval = time.Nanosecond
	require.NotPanics(t, func() {
		storer(b, s)
	})
}
