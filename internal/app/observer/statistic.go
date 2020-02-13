// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package observer

import (
	"github.com/insolar/insolar/insolar"
)

type Statistic struct {
	Pulse     insolar.PulseNumber
	Transfers int
	Nodes     int
}
