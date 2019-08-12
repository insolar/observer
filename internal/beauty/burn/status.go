package burn

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	PENDING  = "PENDING"
	SUCCESS  = "SUCCESS"
	CANCELED = "CANCELED"
)

type addressResult struct {
	status  string
	address string
}

func wastedAddress(payload []byte) addressResult {
	rets := parsePayload(payload)
	if len(rets) < 2 {
		return addressResult{"NOT_ENOUGH_PAYLOAD_PARAMS", ""}
	}

	if rets[1] != nil {
		if retError, ok := rets[1].(map[string]interface{}); ok {
			if val, ok := retError["S"]; ok {
				if msg, ok := val.(string); ok {
					log.Debug(errors.New(msg))
				}
			}
			return addressResult{CANCELED, ""}
		}
		log.Error(errors.New("invalid error value in GetMigrationAddress payload"))
		return addressResult{"INVALID_ERROR_VALUE", ""}
	}
	address, ok := rets[0].(string)
	if !ok {
		return addressResult{"FIRST_PARAM_NOT_STRING", ""}
	}
	return addressResult{SUCCESS, address}
}
