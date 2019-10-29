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

package models

import (
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransactionColumns(t *testing.T) {
	tx := Transaction{}
	v := reflect.TypeOf(tx)
	var fieldList []string

	// First field is tableName
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Name == "ID" || v.Field(i).Name == "tableName" {
			continue
		}
		tag := v.Field(i).Tag.Get("sql")
		// Skip if tag is not defined or ignored
		if tag == "" || tag == "-" {
			continue
		}
		fieldList = append(fieldList, tag)
	}

	columnList := TransactionColumns()

	sort.Strings(columnList)
	sort.Strings(fieldList)

	require.Equal(t, fieldList, columnList)
}
