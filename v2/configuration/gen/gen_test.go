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

package main

import (
	"os"
	"testing"

	"github.com/insolar/observer/internal/configuration"
	"github.com/stretchr/testify/require"
)

func Test_main(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		main()

		require.True(t, fileExists(configuration.ConfigFilePath))
		require.NoError(t, deleteFile(configuration.ConfigFilePath))
	})
	t.Run("failed", func(t *testing.T) {
		_, err := os.Create(configuration.ConfigFilePath)
		require.NoError(t, err)
		err = os.Chmod(configuration.ConfigFilePath, 0444)
		require.NoError(t, err)

		main()

		require.True(t, fileExists(configuration.ConfigFilePath))
		require.NoError(t, deleteFile(configuration.ConfigFilePath))
	})
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func deleteFile(filename string) error {
	return os.Remove(filename)
}
