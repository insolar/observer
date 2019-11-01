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

package xnscoinstats

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type StatsModel struct {
	tableName struct{} `sql:"xns_coin_stats"` //nolint: unused,structcheck

	ID          uint64    `sql:"id,pk"`
	Created     time.Time `sql:"created default:now(),notnull"`
	Total       string    `sql:"total"`
	Max         string    `sql:"max"`
	Circulating string    `sql:"circulating"`
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

type StatsManager struct {
	log        *logrus.Logger
	repository StatsRepo
}

func NewStatsManager(log *logrus.Logger, r StatsRepo) *StatsManager {
	return &StatsManager{
		log:        log,
		repository: r,
	}
}

func (s *StatsManager) Coins() (XnsCoinStats, error) {
	lastStats, err := s.repository.LastStats()
	if err != nil {
		return XnsCoinStats{}, errors.Wrap(err, "failed request get stats")
	}
	// todo make this more clear
	return XnsCoinStats{
		Created:     lastStats.Created,
		Total:       PaddedNumber(lastStats.Total),
		Max:         PaddedNumber(lastStats.Max),
		Circulating: PaddedNumber(lastStats.Circulating),
	}, nil
}

func PaddedNumber(number string) string {
	if len(number) < 10 {
		return fmt.Sprintf("0.%010v", number)
	}

	return number[:len(number)-10] + "." + number[len(number)-10:]
}

func (s *StatsManager) Total() (string, error) {
	res, err := s.Coins()
	if err != nil {
		return "", err
	}
	return res.Total, nil
}

func (s *StatsManager) Max() (string, error) {
	res, err := s.Coins()
	if err != nil {
		return "", err
	}
	return res.Max, nil
}

func (s *StatsManager) Circulating() (string, error) {
	res, err := s.Coins()
	if err != nil {
		return "", err
	}
	return res.Circulating, nil
}

func (s *StatsManager) CountStats() (XnsCoinStats, error) {
	return s.repository.CountStats()
}

func (s *StatsManager) InsertStats(xcs XnsCoinStats) error {
	return s.repository.InsertStats(xcs)
}
