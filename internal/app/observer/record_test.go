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

package observer

import (
	"testing"

	"github.com/insolar/insolar/insolar/record"
	"github.com/stretchr/testify/require"
)

func TestRecord_Marshal(t *testing.T) {
	original := &record.Material{}
	expectedBytes, err := original.Marshal()
	require.NoError(t, err)

	rec := (*Record)(original)
	actualBytes, err := rec.Marshal()

	require.NoError(t, err)
	require.Equal(t, expectedBytes, actualBytes)
}

func TestRecord_Unmarshal(t *testing.T) {
	original := &record.Material{}
	bytes, err := original.Marshal()
	require.NoError(t, err)

	rec := &Record{}
	err = rec.Unmarshal(bytes)
	actual := (*record.Material)(rec)

	require.NoError(t, err)
	require.Equal(t, original, actual)
}
