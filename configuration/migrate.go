// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package configuration

type Migrate struct {
	DB DB
}

func (Migrate) Default() Migrate {
	return Migrate{DB: Observer{}.Default().DB}
}
