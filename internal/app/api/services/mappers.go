// +build !node

package services

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
