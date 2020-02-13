// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package configuration

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
		require.NotContains(t, replacePassword(with), password)
	})

	t.Run("not_replaced", func(t *testing.T) {
		require.NotContains(t, without, password)
		require.NotContains(t, replacePassword(without), password)
		require.Equal(t, without, replacePassword(without))
	})
}
