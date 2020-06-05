// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

// +build !node

package api

import (
	"fmt"

	"github.com/insolar/observer/internal/models"
)

func (response *ResponsesMarketStatsYaml) addHistoryPoints(points []models.PriceHistory) {
	var parsedPoints []struct {
		Price     string `json:"price"`
		Timestamp int64  `json:"timestamp"`
	}
	for _, point := range points {
		parsedPoints = append(parsedPoints, struct {
			Price     string `json:"price"`
			Timestamp int64  `json:"timestamp"`
		}{
			fmt.Sprintf("%v", point.Price),
			point.Timestamp.Unix(),
		})
	}
	response.PriceHistory = &parsedPoints
}
