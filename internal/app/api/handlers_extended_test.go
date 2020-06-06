// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

// +build !node

package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/models"
)

func TestObserverServer_SupplyStatsEmpty(t *testing.T) {
	resp, err := http.Get("http://" + apihost + "/api/stats/supply/total")
	require.NoError(t, err)
	require.True(t, resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK)
}

func TestObserverServer_SupplyStats(t *testing.T) {
	total := "1111111111111"
	totalr := "111.1111111111"

	coins := models.SupplyStats{
		Created: time.Now(),
		Total:   total,
	}

	err := db.Insert(&coins)
	require.NoError(t, err)

	resp, err := http.Get("http://" + apihost + "/api/stats/supply/total")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, totalr, string(bodyBytes))
}

func TestFee(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/fee/123")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		received := ResponsesFeeYaml{}
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Equal(t, testFee.String(), received.Fee)
	})

	t.Run("uuid", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/fee/31f277c7-67f8-45b5-ae26-ff127d62a9ba")
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		received := ResponsesInvalidAmountYaml{}
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Equal(t, []string{"invalid amount"}, received.Error)
	})

	t.Run("negative", func(t *testing.T) {
		resp, err := http.Get("http://" + apihost + "/api/fee/-1")
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		received := ResponsesInvalidAmountYaml{}
		err = json.Unmarshal(bodyBytes, &received)
		require.NoError(t, err)
		require.Equal(t, []string{"negative amount"}, received.Error)
	})
}

func TestObserverServer_Notifications(t *testing.T) {
	apiUrl := "http://" + apihost + "/api/notification"

	// No content status is displayed
	if _, err := db.Exec("DELETE FROM notifications"); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(apiUrl)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Present notification is published
	presentNotificationName := uuid.New().String()
	presentNotification := &models.Notification{
		Message: presentNotificationName,
		Start:   time.Now().Add(-3 * time.Hour),
		Stop:    time.Now().Add(3 * time.Hour),
	}
	err = db.Insert(presentNotification)
	require.NoError(t, err)

	resp, err = http.Get(apiUrl)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	jsonResp := ResponsesNotificationInfoYaml{}
	err = json.Unmarshal(bodyBytes, &jsonResp)
	require.NoError(t, err)
	require.Equal(t, presentNotificationName, jsonResp.Notification)

	// Past notification was not published
	resp, err = http.Get(apiUrl)
	require.NoError(t, err)

	oldNotificationName := uuid.New().String()
	err = db.Insert(&models.Notification{
		Message: oldNotificationName,
		Start:   time.Now().Add(-10 * time.Hour),
		Stop:    time.Now().Add(-9 * time.Hour),
	})
	require.NoError(t, err)

	resp, err = http.Get(apiUrl)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	jsonResp = ResponsesNotificationInfoYaml{}
	err = json.Unmarshal(bodyBytes, &jsonResp)
	require.NoError(t, err)
	require.NotEqual(t, oldNotificationName, jsonResp.Notification)

	// Future notification is not published yet
	futureNotificationName := uuid.New().String()
	err = db.Insert(&models.Notification{
		Message: futureNotificationName,
		Start:   time.Now().Add(20 * time.Hour),
		Stop:    time.Now().Add(24 * time.Hour),
	})
	require.NoError(t, err)

	resp, err = http.Get(apiUrl)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	jsonResp = ResponsesNotificationInfoYaml{}
	err = json.Unmarshal(bodyBytes, &jsonResp)
	require.NoError(t, err)
	require.NotEqual(t, futureNotificationName, jsonResp.Notification)
}
