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

func parseAddBurnAddressesCallParams(request member.Request) []string {
	callParams, ok := request.Params.CallParams.(map[string]interface{})
	if !ok {
		log.Warnf("failed to cast CallParams to map[string]interface{}")
		return []string{}
	}
	burnAddresses, ok := callParams["burnAddresses"]
	if !ok {
		log.Warnf(`failed to get CallParams["burnAddresses"]`)
		return []string{}
	}

	interfaces, ok := burnAddresses.([]interface{})
	if !ok {
		log.Warnf(`failed to cast CallParams["burnAddresses"] to []interface{}`)
		return []string{}
	}
	addresses := []string{}
	for _, a := range interfaces {
		if v, ok := a.(string); ok {
			addresses = append(addresses, v)
		} else {
			log.Warnf(`failed to cast CallParams["burnAddresses"][i] to string`)
			return []string{}
		}
	}
	return addresses
}
