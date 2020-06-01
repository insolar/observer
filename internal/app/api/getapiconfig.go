// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

// +build !extended

package api

import (
	"github.com/insolar/observer/configuration"
)

func getApiConfig() configuration.ApiConfig {
	return configuration.Api{}
}
