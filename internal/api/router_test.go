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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRouter(t *testing.T) {
	require.NotNil(t, NewRouter())
}

func TestRouter_Init(t *testing.T) {
	router := NewRouter()
	ctx := context.Background()

	require.NoError(t, router.Init(ctx))
}

func TestRouter_Start(t *testing.T) {
	router := NewRouter()
	ctx := context.Background()

	// TODO: check before init panic in goroutine (smth like):
	// t.Run("before_init", func(t *testing.T) {
	// 	require.Panics(t, func() { router.Start(ctx) })
	// })

	t.Run("after_init", func(t *testing.T) {
		assert.NoError(t, router.Init(ctx))
		require.NoError(t, router.Start(ctx))
	})
}

func TestRouter_Stop(t *testing.T) {
	router := NewRouter()
	ctx := context.Background()

	t.Run("after_start", func(t *testing.T) {
		assert.NoError(t, router.Init(ctx))
		assert.NoError(t, router.Start(ctx))
		require.NoError(t, router.Stop(ctx))
	})
}
