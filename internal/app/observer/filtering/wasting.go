// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package filtering

import (
	"github.com/insolar/observer/internal/app/observer"
)

type WastingFilter struct{}

func NewWastingFilter() *WastingFilter {
	return &WastingFilter{}
}

func (*WastingFilter) Filter(wastings map[string]*observer.Wasting, addresses map[string]*observer.MigrationAddress) {
	// We try to apply migration address wasting in memory.
	for key, wasting := range wastings {
		addr, ok := addresses[wasting.Addr]
		if !ok {
			continue
		}
		addr.Wasted = true
		delete(wastings, key)
	}
}
