package observer

import (
	"github.com/insolar/insolar/insolar"
)

type Statistic struct {
	Pulse     insolar.PulseNumber
	Transfers int
	Nodes     int
}
