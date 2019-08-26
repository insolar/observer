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

package migration

import (
	"github.com/insolar/insolar/logicrunner/builtin/contract/member"
	log "github.com/sirupsen/logrus"
)

func parseAddMigrationAddressesCallParams(request member.Request) []string {
	callParams, ok := request.Params.CallParams.(map[string]interface{})
	if !ok {
		log.Warnf("failed to cast CallParams to map[string]interface{}")
		return []string{}
	}
	migrationAddresses, ok := callParams["migrationAddresses"]
	if !ok {
		log.Warnf(`failed to get CallParams["migrationAddresses"]`)
		return []string{}
	}

	interfaces, ok := migrationAddresses.([]interface{})
	if !ok {
		log.Warnf(`failed to cast CallParams["migrationAddresses"] to []interface{}`)
		return []string{}
	}
	addresses := []string{}
	for _, a := range interfaces {
		if v, ok := a.(string); ok {
			addresses = append(addresses, v)
		} else {
			log.Warnf(`failed to cast CallParams["migrationAddresses"][i] to string`)
			return []string{}
		}
	}
	return addresses
}
