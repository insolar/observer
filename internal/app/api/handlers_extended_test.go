// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

// +build !node

package api

import (
	"bytes"
	"context"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/secrets"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	crypto "github.com/insolar/x-crypto"
	"github.com/insolar/x-crypto/ecdsa"
	"github.com/insolar/x-crypto/rand"
	"github.com/insolar/x-crypto/sha256"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/component"
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

const (
	Digest      = "Digest"
	Signature   = "Signature"
	ContentType = "Content-Type"
)

func TestSetAugmentedAddress(t *testing.T) {
	defer truncateDB(t)
	member1 := gen.Reference()
	member2 := gen.Reference()

	privateKey, err := secrets.GeneratePrivateKeyEthereum()
	publicKeyPEM, err := secrets.ExportPublicKeyPEM(secrets.ExtractPublicKey(privateKey))
	publicKey := string(publicKeyPEM)
	canonicalPk, err := foundation.ExtractCanonicalPublicKey(publicKey)

	insertMember(t, member1, nil, nil, "0", canonicalPk)
	insertMember(t, member2, nil, nil, "0", randomString())

	member1Str := url.QueryEscape(member1.String())
	body := SetAugmentedAddressJSONRequestBody{
		AugmentedAddress: "0xF4e1507486dFE411785B00d7D00A1f1a484f00E6",
		PublicKey:        publicKey,
	}
	req := signedRequest(t, body, member1Str, privateKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	var setResp ResponsesAugmentedAddressYaml
	err = json.Unmarshal(bodyBytes, &setResp)
	require.NoError(t, err)
	require.Equal(t, body.AugmentedAddress, setResp.AugmentedAddress)

	addrInDB, err := component.GetAugmentedAddress(context.Background(), db, member1.Bytes())
	require.NoError(t, err)
	require.Equal(t, body.AugmentedAddress, addrInDB.Address)
}

func TestGetAugmentedAddress(t *testing.T) {
	defer truncateDB(t)
	member1 := gen.Reference()
	addr := insertAugmentedAddress(t, member1)

	member1Str := url.QueryEscape(member1.String())
	req, err := http.NewRequest("GET", "http://"+apihost+"/api/member/augmentedAddress/"+member1Str, nil)
	require.NoError(t, err)
	req.Header.Set(ContentType, "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	var getResp ResponsesAugmentedAddressYaml
	err = json.Unmarshal(bodyBytes, &getResp)
	require.NoError(t, err)
	require.Equal(t, addr, getResp.AugmentedAddress)
}

func TestGetAugmentedAddress_Empty(t *testing.T) {
	defer truncateDB(t)
	member1 := gen.Reference()

	member1Str := url.QueryEscape(member1.String())
	req, err := http.NewRequest("GET", "http://"+apihost+"/api/member/augmentedAddress/"+member1Str, nil)
	require.NoError(t, err)
	req.Header.Set(ContentType, "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	var getResp ResponsesAugmentedAddressYaml
	err = json.Unmarshal(bodyBytes, &getResp)
	require.NoError(t, err)
	require.Equal(t, "", getResp.AugmentedAddress)
}

func TestSetAugmentedAddress_UpdateExisted(t *testing.T) {
	defer truncateDB(t)
	member1 := gen.Reference()
	member2 := gen.Reference()

	privateKey, err := secrets.GeneratePrivateKeyEthereum()
	publicKeyPEM, err := secrets.ExportPublicKeyPEM(secrets.ExtractPublicKey(privateKey))
	publicKey := string(publicKeyPEM)
	canonicalPk, err := foundation.ExtractCanonicalPublicKey(publicKey)

	insertMember(t, member1, nil, nil, "0", canonicalPk)
	insertMember(t, member2, nil, nil, "0", randomString())
	oldAddr := insertAugmentedAddress(t, member1)

	member1Str := url.QueryEscape(member1.String())
	body := SetAugmentedAddressJSONRequestBody{
		AugmentedAddress: "0xF4e1507486dFE411785B00d7D00A1f1a484f00E6",
		PublicKey:        publicKey,
	}
	req := signedRequest(t, body, member1Str, privateKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	var setResp ResponsesAugmentedAddressYaml
	err = json.Unmarshal(bodyBytes, &setResp)
	require.NoError(t, err)
	require.Equal(t, body.AugmentedAddress, setResp.AugmentedAddress)
	require.NotEqual(t, oldAddr, setResp.AugmentedAddress)

	addrInDB, err := component.GetAugmentedAddress(context.Background(), db, member1.Bytes())
	require.NoError(t, err)
	require.Equal(t, body.AugmentedAddress, addrInDB.Address)
}

func TestSetAugmentedAddress_ErrorWrongSign(t *testing.T) {
	defer truncateDB(t)
	member1 := gen.Reference()
	member2 := gen.Reference()

	privateKey, err := secrets.GeneratePrivateKeyEthereum()
	publicKeyPEM, err := secrets.ExportPublicKeyPEM(secrets.ExtractPublicKey(privateKey))
	publicKey := string(publicKeyPEM)
	canonicalPk, err := foundation.ExtractCanonicalPublicKey(publicKey)

	insertMember(t, member1, nil, nil, "0", canonicalPk)
	insertMember(t, member2, nil, nil, "0", randomString())
	oldAddr := insertAugmentedAddress(t, member1)

	member1Str := url.QueryEscape(member1.String())
	body := SetAugmentedAddressJSONRequestBody{
		AugmentedAddress: "0xF4e1507486dFE411785B00d7D00A1f1a484f00E6",
		PublicKey:        publicKey,
	}
	privateKeySecond, err := secrets.GeneratePrivateKeyEthereum()
	req := signedRequest(t, body, member1Str, privateKeySecond)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)

	received := &ErrorMessage{}
	expected := &ErrorMessage{Error: []string{"signature is wrong"}}
	requireEqualResponse(t, resp, received, expected)

	addrInDB, err := component.GetAugmentedAddress(context.Background(), db, member1.Bytes())
	require.NoError(t, err)
	require.Equal(t, oldAddr, addrInDB.Address)
}

func TestSetAugmentedAddress_ErrorInvalidAddress(t *testing.T) {
	defer truncateDB(t)
	member1 := gen.Reference()
	member2 := gen.Reference()

	privateKey, err := secrets.GeneratePrivateKeyEthereum()
	publicKeyPEM, err := secrets.ExportPublicKeyPEM(secrets.ExtractPublicKey(privateKey))
	publicKey := string(publicKeyPEM)
	canonicalPk, err := foundation.ExtractCanonicalPublicKey(publicKey)

	insertMember(t, member1, nil, nil, "0", canonicalPk)
	insertMember(t, member2, nil, nil, "0", randomString())
	oldAddr := insertAugmentedAddress(t, member1)

	member1Str := url.QueryEscape(member1.String())
	body := SetAugmentedAddressJSONRequestBody{
		AugmentedAddress: "not_valid",
		PublicKey:        publicKey,
	}
	req := signedRequest(t, body, member1Str, privateKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	received := &ErrorMessage{}
	expected := &ErrorMessage{Error: []string{"invalid metamask address"}}
	requireEqualResponse(t, resp, received, expected)

	addrInDB, err := component.GetAugmentedAddress(context.Background(), db, member1.Bytes())
	require.NoError(t, err)
	require.Equal(t, oldAddr, addrInDB.Address)
}

func TestSetAugmentedAddress_ErrorEmptySign(t *testing.T) {
	defer truncateDB(t)
	member1 := gen.Reference()

	privateKey, err := secrets.GeneratePrivateKeyEthereum()
	publicKeyPEM, err := secrets.ExportPublicKeyPEM(secrets.ExtractPublicKey(privateKey))
	publicKey := string(publicKeyPEM)

	member1Str := url.QueryEscape(member1.String())
	body := SetAugmentedAddressJSONRequestBody{
		AugmentedAddress: "not_valid",
		PublicKey:        publicKey,
	}
	jsonValue, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "http://"+apihost+"/api/member/augmentedAddress/"+member1Str, bytes.NewReader(jsonValue))
	require.NoError(t, err)
	req.Header.Set(ContentType, "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	_, err = component.GetAugmentedAddress(context.Background(), db, member1.Bytes())
	require.Equal(t, component.ErrReferenceNotFound, err)
}

func sign(privateKey crypto.PrivateKey, data []byte) (string, error) {
	hash := sha256.Sum256(data)

	r, s, err := ecdsa.Sign(rand.Reader, privateKey.(*ecdsa.PrivateKey), hash[:])
	if err != nil {
		return "", errors.Wrap(err, "[ sign ] Cant sign data")
	}

	return marshalSig(r, s)
}

// marshalSig encodes ECDSA signature to ASN.1.
func marshalSig(r, s *big.Int) (string, error) {
	var ecdsaSig struct {
		R, S *big.Int
	}
	ecdsaSig.R, ecdsaSig.S = r, s

	asnSig, err := asn1.Marshal(ecdsaSig)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(asnSig), nil
}

func signedRequest(t *testing.T, body SetAugmentedAddressJSONRequestBody, ref string, privateKey crypto.PrivateKey) *http.Request {
	jsonValue, err := json.Marshal(body)
	require.NoError(t, err)

	signature, err := sign(privateKey, jsonValue)
	require.NoError(t, err)

	h := sha256.New()
	_, err = h.Write(jsonValue)
	require.NoError(t, err)
	sha := base64.StdEncoding.EncodeToString(h.Sum(nil))

	req, err := http.NewRequest("POST", "http://"+apihost+"/api/member/augmentedAddress/"+ref, bytes.NewReader(jsonValue))
	require.NoError(t, err)

	req.Header.Set(Digest, "SHA-256="+sha)
	req.Header.Set(Signature, "keyId=\"member-pub-key\", algorithm=\"ecdsa\", headers=\"digest\", signature="+signature)
	req.Header.Set(ContentType, "application/json")

	return req
}

func insertAugmentedAddress(t *testing.T, reference insolar.Reference) string {
	addr := models.AugmentedAddress{
		Reference: reference.Bytes(),
		Address:   randomString(),
	}
	err := db.Insert(&addr)
	require.NoError(t, err)
	return addr.Address
}
