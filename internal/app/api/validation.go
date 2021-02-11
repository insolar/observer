// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/observer/blob/master/LICENSE.md.

// +build !node

package api

import (
	"bytes"
	"encoding/base64"
	"strings"

	"github.com/insolar/x-crypto/sha256"
	"github.com/pkg/errors"
)

func validateRequestHeaders(digest string, signature string, body []byte) (string, error) {
	// Digest = "SHA-256=<hashString>"
	// Signature = "keyId="member-pub-key", algorithm="ecdsa", headers="digest", signature=<signatureString>"
	if len(digest) < 15 || strings.Count(digest, "=") < 2 || len(signature) == 15 ||
		strings.Count(signature, "=") < 4 || len(body) == 0 {
		return "", errors.Errorf("invalid input data length digest: %d, signature: %d, body: %d", len(digest),
			len(signature), len(body))
	}
	h := sha256.New()
	_, err := h.Write(body)
	if err != nil {
		return "", errors.Wrap(err, "cant calculate hash")
	}
	calculatedHash := h.Sum(nil)
	digest, err = parseDigest(digest)
	if err != nil {
		return "", err
	}
	incomingHash, err := base64.StdEncoding.DecodeString(digest)
	if err != nil {
		return "", errors.Wrap(err, "cant decode digest")
	}

	if !bytes.Equal(calculatedHash, incomingHash) {
		return "", errors.New("incorrect digest")
	}

	signature, err = parseSignature(signature)
	if err != nil {
		return "", err
	}
	return signature, nil
}

func parseDigest(digest string) (string, error) {
	index := strings.IndexByte(digest, '=')
	if index < 1 || (index+1) >= len(digest) {
		return "", errors.New("invalid digest")
	}

	return digest[index+1:], nil
}

func parseSignature(signature string) (string, error) {
	index := strings.Index(signature, "signature=")
	if index < 1 || (index+10) >= len(signature) {
		return "", errors.New("invalid signature")
	}

	return signature[index+10:], nil
}
