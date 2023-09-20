package component

import (
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer/postgres"
	"github.com/insolar/observer/internal/app/observer/store"
	"github.com/insolar/observer/observability"
)

func makeInitter(cfg *configuration.Observer, obs *observability.Observability, conn PGer) func() *state {
	logger := obs.Log()
	last := MustKnowPulse(obs, conn.PG())
	metricState := getMetricState(cfg, obs, conn.PG())
	st := state{
		last: last,
		ms:   metricState,
	}
	logger.Debugf("State restored: %+v", st)
	logger.Debugf("Metric state restored : %+v", metricState)
	return func() *state {
		return &st
	}
}

func MustKnowPulse(obs *observability.Observability, db orm.DB) insolar.PulseNumber {
	pulses := postgres.NewPulseStorage(obs.Log(), db)
	p, err := pulses.Last()
	if err == store.ErrNotFound {
		return 0
	}
	if err != nil {
		panic(errors.Wrap(err, "Something wrong with pulses in DB or DB itself"))
	}
	return p.Number
}

type metricState struct {
	totalVesting            int
	totalMigrationAddresses int
}

func (ms *metricState) Reset() {
	ms.totalVesting = 0
	ms.totalMigrationAddresses = 0
}

func getMetricState(cfg *configuration.Observer, obs *observability.Observability, db orm.DB) metricState {
	ma := postgres.NewMigrationAddressStorage(&cfg.DB, obs, db)
	return metricState{
		totalVesting:            ma.Wasted(),
		totalMigrationAddresses: ma.TotalMigrationAddresses(),
	}
}
