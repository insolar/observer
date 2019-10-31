package xnscoinstats

import (
	"time"

	"github.com/go-pg/pg"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const TenBillion uint64 = 10000000000

type StatsModel struct {
	tableName struct{} `sql:"xns_coin_stats"` //nolint: unused,structcheck

	ID          uint64    `sql:",pk"`
	Created     time.Time `sql:"default:now(),notnull"`
	Total       string    `sql:""`
	Max         string    `sql:""`
	Circulating string    `sql:""`
}

type XnsCoinStats struct {
	Created     time.Time `json:"-"`
	Total       string    `json:"total"`
	Max         string    `json:"max"`
	Circulating string    `json:"circulating"`
}

type StatsGetter interface {
	Coins() (XnsCoinStats, error)
	Total() (string, error)
	Max() (string, error)
	Circulating() (string, error)
}

type StatsSetter interface {
	SetStats(stats XnsCoinStats) error
}

type StatsCounter interface {
	CountStats() (XnsCoinStats, error)
}

type StatsRepository struct {
	db  *pg.DB
	log *logrus.Logger
}

func NewStatsRepository(db *pg.DB, log *logrus.Logger) *StatsRepository {
	return &StatsRepository{
		db:  db,
		log: log,
	}
}
func (s *StatsRepository) Coins() (XnsCoinStats, error) {
	lastStats := &StatsModel{}
	err := s.db.Model(lastStats).Last()
	if err != nil && err != pg.ErrNoRows {
		s.log.Error(errors.Wrapf(err, "failed request to db"))
		return XnsCoinStats{}, errors.Wrap(err, "failed request to db")
	}
	if err == pg.ErrNoRows {
		return XnsCoinStats{}, errors.New("No data")
	}
	// todo make this more obvious
	return XnsCoinStats{
		Created:     lastStats.Created,
		Total:       lastStats.Total[:len(lastStats.Total)-10] + "." + lastStats.Total[len(lastStats.Total)-10:],
		Max:         lastStats.Max[:len(lastStats.Total)-10] + "." + lastStats.Max[len(lastStats.Total)-10:],
		Circulating: lastStats.Circulating[:len(lastStats.Total)-10] + "." + lastStats.Circulating[len(lastStats.Total)-10:],
	}, nil
}

func (s *StatsRepository) Total() (string, error) {
	res, err := s.Coins()
	if err != nil {
		return "", err
	}
	return res.Total, nil
}

func (s *StatsRepository) Max() (string, error) {
	res, err := s.Coins()
	if err != nil {
		return "", err
	}
	return res.Max, nil
}

func (s *StatsRepository) Circulating() (string, error) {
	res, err := s.Coins()
	if err != nil {
		return "", err
	}
	return res.Circulating, nil
}

func (s *StatsRepository) CountStats() (XnsCoinStats, error) {
	sql := `
WITH stats as (SELECT d.amount::numeric(24),
                      d.hold_release_date,
                      round(extract(epoch from now()))      as pulse_ts,
                      d.hold_release_date + 365 * 24 * 3600 as vesting_ends_ts
               from deposits d),
     sums as (
         SELECT sum(floor(stats.amount * least(greatest(stats.pulse_ts - stats.hold_release_date, 0) /
                                               (stats.vesting_ends_ts - stats.hold_release_date), 1)))      as free
         from stats),
     circulating as (select sum(balance::numeric(24)) as sum from members),
     dep_balance as (select sum(balance::numeric(24)) as sum from deposits)

SELECT
       circulating.sum::numeric                   as circulating,
       (circulating.sum + free)::numeric            as total,
       (circulating.sum + dep_balance.sum)::numeric as max
from sums,
     circulating,
     dep_balance
;
`

	stats := XnsCoinStats{}
	_, err := s.db.Query(&stats, sql)
	if err != nil {
		return XnsCoinStats{}, errors.Wrap(err, "failed request to db")
	}
	s.log.Debugf("XnsCoinStats: %+v", stats)
	return stats, nil
}

func (s *StatsRepository) InsertStats(xcs XnsCoinStats) error {
	stats := StatsModel{
		Created:     time.Now(),
		Total:       xcs.Total,
		Max:         xcs.Max,
		Circulating: xcs.Circulating,
	}

	err := s.db.Insert(&stats)
	if err != nil {
		return errors.Wrap(err, "failed to insert stats")
	}

	return nil
}
