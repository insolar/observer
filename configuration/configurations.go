// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

// +build !extended

package configuration

func Configurations() map[string]interface{} {
	cfgs := make(map[string]interface{})
	cfgs["observerapi.yaml"] = API{}.Default()
	cfgs["observer.yaml"] = Observer{}.Default()
	cfgs["migrate.yaml"] = Migrate{}.Default()

	return cfgs
}
