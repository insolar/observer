// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

package postgres

import (
	"time"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/models"
)

//go:generate minimock -i NetworkStatsRepo -o ./ -s _mock.go -g

type NetworkStatsRepo interface {
	LastStats() (models.NetworkStats, error)
	InsertStats(models.NetworkStats) error
	CountStats() (models.NetworkStats, error)
}

type NetworkStatsRepository struct {
	db orm.DB
}

func NewNetworkStatsRepository(db orm.DB) NetworkStatsRepo {
	return &NetworkStatsRepository{db: db}
}

func (s *NetworkStatsRepository) LastStats() (models.NetworkStats, error) {
	lastStats := &models.NetworkStats{}
	err := s.db.Model(lastStats).Last()
	if err != nil && err != pg.ErrNoRows {
		return models.NetworkStats{}, errors.Wrap(err, "failed request to db")
	}
	if err == pg.ErrNoRows {
		return models.NetworkStats{}, errors.New("No data")
	}
	return *lastStats, nil
}

func (s *NetworkStatsRepository) InsertStats(xcs models.NetworkStats) error {
	err := s.db.Insert(&xcs)
	if err != nil {
		return errors.Wrap(err, "failed to insert stats")
	}

	return nil
}

func (s *NetworkStatsRepository) CountStats() (models.NetworkStats, error) {
	stats := models.NetworkStats{
		Created: time.Now(),
	}

	// LastPulseNumber and Nodes
	{
		pulseSchema := models.Pulse{}
		err := s.db.Model(&pulseSchema).
			Order("pulse DESC").
			Limit(1).
			Select()
		if err == pg.ErrNoRows {
			pulseSchema.Pulse = uint32(insolar.GenesisPulse.PulseNumber)
		} else if err != nil {
			return models.NetworkStats{}, errors.Wrap(err, "couldn't get last pulse data")
		}
		stats.PulseNumber = int(pulseSchema.Pulse)
		stats.Nodes = int(pulseSchema.Nodes)
	}

	// MonthTransactions
	{
		monthAgoPulse := uint32(insolar.GenesisPulse.PulseNumber)
		pulseSchema := models.Pulse{}
		err := s.db.Model(&pulseSchema).
			Where("pulse_date < extract(epoch from (NOW() - INTERVAL '30 DAYS'))").
			Order("pulse DESC").
			Limit(1).
			Select()
		if err != nil && err != pg.ErrNoRows {
			return models.NetworkStats{}, errors.Wrap(err, "couldn't get last pulse data")
		} else if err == nil {
			monthAgoPulse = pulseSchema.Pulse
		}

		sqlRes := struct{ Count int }{}
		_, err = s.db.QueryOne(&sqlRes, "SELECT COUNT(1) AS Count FROM simple_transactions"+
			" WHERE finish_pulse_record[1] >= ? AND status_registered", monthAgoPulse)
		if err != nil {
			return models.NetworkStats{}, errors.Wrap(err, "couldn't count total transactions")
		}
		stats.MonthTransactions = sqlRes.Count
	}

	// TotalTransactions
	{
		sqlRes := struct{ Count int }{}
		_, err := s.db.QueryOne(
			&sqlRes,
			"SELECT COUNT(1) AS Count FROM simple_transactions WHERE status_registered",
		)
		if err != nil {
			return models.NetworkStats{}, errors.Wrap(err, "couldn't count total transactions")
		}
		stats.TotalTransactions = sqlRes.Count
	}

	// TotalAccounts
	{
		sqlRes := struct{ Count int }{}
		_, err := s.db.QueryOne(&sqlRes, "SELECT COUNT(1) AS Count FROM members")
		if err != nil {
			return models.NetworkStats{}, errors.Wrap(err, "couldn't count total accounts")
		}
		stats.TotalAccounts = sqlRes.Count
	}

	// MaxTPS
	{
		sqlRes := struct{ Count int }{}
		sql := "SELECT MAX(t.tpp) AS Count FROM (" +
			"  SELECT COUNT(1) as tpp FROM simple_transactions" +
			"  WHERE status_registered AND finish_pulse_record IS NOT NULL" +
			"  GROUP BY finish_pulse_record[1]" +
			") AS t"
		_, err := s.db.QueryOne(&sqlRes, sql)
		if err != nil {
			return models.NetworkStats{}, errors.Wrap(err, "failed request to db")
		}
		stats.MaxTPS = sqlRes.Count
	}

	// CurrentTPS
	{
		sqlRes := struct{ Count int }{}
		sql := "SELECT COUNT(1) AS Count FROM simple_transactions" +
			" WHERE finish_pulse_record[1] = (" +
			"   select pulse from pulses ORDER BY pulse DESC LIMIT 1" +
			" ) AND status_registered"
		_, err := s.db.QueryOne(&sqlRes, sql)
		if err != nil {
			return models.NetworkStats{}, errors.Wrap(err, "failed request to db")
		}
		stats.CurrentTPS = sqlRes.Count
	}

	return stats, nil
}
