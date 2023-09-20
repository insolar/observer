package postgres

import (
	"fmt"
	"time"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/models"
)

var ErrNoStats = errors.New("No stats")

type SupplyStatsRepository struct {
	db orm.DB
}

func NewSupplyStatsRepository(db orm.DB) *SupplyStatsRepository {
	return &SupplyStatsRepository{db: db}
}

func (s *SupplyStatsRepository) LastStats() (models.SupplyStats, error) {
	sql := fmt.Sprintf(`select * from supply_stats where now() > created order by created DESC limit 1`)
	lastStats := &models.SupplyStats{}
	_, err := s.db.QueryOne(lastStats, sql)
	if err != nil && err != pg.ErrNoRows {
		return models.SupplyStats{}, errors.Wrap(err, "failed request to db")
	}
	if err == pg.ErrNoRows {
		return models.SupplyStats{}, ErrNoStats
	}
	return *lastStats, nil
}

func (s *SupplyStatsRepository) CountStats() (models.SupplyStats, error) {
	sql := fmt.Sprintf(`SELECT (SELECT coalesce(sum(balance::numeric(24)), 0) FROM members) + 
								(SELECT coalesce(sum(balance::numeric(24)), 0) FROM deposits) AS total;`)
	stats := models.SupplyStats{}
	_, err := s.db.Query(&stats, sql)
	if err != nil {
		return models.SupplyStats{}, errors.Wrap(err, "failed request to db")
	}
	return stats, nil
}

func (s *SupplyStatsRepository) InsertStats(stats models.SupplyStats) error {
	stats.Created = time.Now()

	err := s.db.Insert(&stats)
	if err != nil {
		return errors.Wrap(err, "failed to insert stats")
	}

	return nil
}
