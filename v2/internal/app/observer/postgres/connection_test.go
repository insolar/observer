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

package postgres

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/v2/configuration"
)

func TestNewConnectionHolder(t *testing.T) {
	cfg := &configuration.Configuration{}
	cfg.DB.URL = "invalid url"
	require.Nil(t, NewConnectionHolder(cfg))
}

func TestConnectionHolder_DB(t *testing.T) {
	cfg := configuration.Default()
	holder := NewConnectionHolder(cfg)

	db := holder.DB()
	require.NotNil(t, db)
}

func TestConnectionHolder_Close(t *testing.T) {
	cfg := configuration.Default()
	holder := NewConnectionHolder(cfg)

	err := holder.Close()
	require.NoError(t, err)

	err = holder.Close()
	require.NoError(t, err)
}
