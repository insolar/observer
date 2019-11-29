//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package component

import (
	"fmt"
	"time"

	"github.com/go-pg/pg/orm"
	"github.com/pkg/errors"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/observer/internal/app/observer/postgres"
)

type XnsCoinStats interface {
	Created() time.Time
	Total() string
	Max() string
	Circulating() string
}

type XnsCoinData struct {
	created     time.Time
	total       string
	max         string
	circulating string
}

func (s XnsCoinData) Created() time.Time {
	return s.created
}

func (s XnsCoinData) Total() string {
	return s.total
}

func (s XnsCoinData) Max() string {
	return s.max
}

func (s XnsCoinData) Circulating() string {
	return s.circulating
}

type StatsGetter interface {
	Supply() (XnsCoinStats, error)
	Total() (string, error)
	Max() (string, error)
	Circulating() (string, error)
}

type StatsCollecter interface {
	CountStats(time *time.Time) (XnsCoinData, error)
	InsertStats(xcs XnsCoinData) error
}

type StatsManager struct {
	log        insolar.Logger
	repository postgres.SupplyStatsRepo
}

func NewStatsManager(log insolar.Logger, r postgres.SupplyStatsRepo) *StatsManager {
	return &StatsManager{
		log:        log,
		repository: r,
	}
}

func (s *StatsManager) Supply() (XnsCoinStats, error) {
	lastStats, err := s.repository.LastStats()
	if err != nil {
		return XnsCoinData{}, errors.Wrap(err, "failed request get stats")
	}
	return &XnsCoinData{
		created:     lastStats.Created,
		total:       lastStats.Total,
		max:         lastStats.Max,
		circulating: lastStats.Circulating,
	}, nil
}

func (s *StatsManager) Total() (string, error) {
	res, err := s.Supply()
	if err != nil {
		return "", err
	}
	return s.convertToCMCFormat(res.Total()), nil
}

func (s *StatsManager) Max() (string, error) {
	res, err := s.Supply()
	if err != nil {
		return "", err
	}
	return s.convertToCMCFormat(res.Max()), nil
}

func (s *StatsManager) Circulating() (string, error) {
	res, err := s.Supply()
	if err != nil {
		return "", err
	}
	return s.convertToCMCFormat(res.Circulating()), nil
}

func (s *StatsManager) CountStats(time *time.Time) (XnsCoinData, error) {
	st, err := s.repository.CountStats(time)
	if err != nil {
		return XnsCoinData{}, err
	}
	return s.toDTO(st), nil
}

func (s *StatsManager) InsertStats(xcs XnsCoinData) error {
	return s.repository.InsertStats(s.fromDTO(xcs))
}

func (s *StatsManager) toDTO(stats postgres.SupplyStatsModel) XnsCoinData {
	return XnsCoinData{
		created:     stats.Created,
		total:       stats.Total,
		max:         stats.Max,
		circulating: stats.Circulating,
	}
}

func (s *StatsManager) fromDTO(stats XnsCoinData) postgres.SupplyStatsModel {
	return postgres.SupplyStatsModel{
		Created:     stats.Created(),
		Total:       stats.Total(),
		Max:         stats.Max(),
		Circulating: stats.Circulating(),
	}
}

func (s *StatsManager) convertToCMCFormat(str string) string {
	if len(str) <= 10 {
		return fmt.Sprintf("0.%010s", str)
	}
	return str[:len(str)-10] + "." + str[len(str)-10:]
}

type CalculateStatsCommand struct {
	log          insolar.Logger
	db           orm.DB
	statsManager StatsCollecter
}

func NewCalculateStatsCommand(logger insolar.Logger, db orm.DB, manager StatsCollecter) *CalculateStatsCommand {
	return &CalculateStatsCommand{
		log:          logger,
		db:           db,
		statsManager: manager,
	}
}

func (s *CalculateStatsCommand) Run(currentDT *time.Time) (XnsCoinStats, error) {
	stats, err := s.statsManager.CountStats(currentDT)
	if err != nil {
		return XnsCoinData{}, err
	}

	s.log.Debugf("Collected stats: %+v", stats)
	// don't save if it is historical request
	if currentDT != nil {
		return stats, nil
	}

	err = s.statsManager.InsertStats(stats)
	if err != nil {
		return stats, err
	}
	return stats, nil
}
