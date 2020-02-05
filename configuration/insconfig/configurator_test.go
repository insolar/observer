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

package insconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_replacePassword(t *testing.T) {
	const password = "super_secret_password"
	const with = "postgresql://observer:" + password + "@127.0.0.1:5432/dev-observer?sslmode=disable"
	const without = "postgres://postgres@localhost/postgres?sslmode=disable"

	t.Run("replaced", func(t *testing.T) {
		require.Contains(t, with, password)
		require.NotContains(t, replaceDBPassword(with), password)
	})

	t.Run("not_replaced", func(t *testing.T) {
		require.NotContains(t, without, password)
		require.NotContains(t, replaceDBPassword(without), password)
		require.Equal(t, without, replaceDBPassword(without))
	})
}
