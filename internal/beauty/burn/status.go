package burn

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
	if retError, ok := rets[1].(error); ok {
		if retError != nil {
			return addressResult{CANCELED, ""}
		}
	}
	address, ok := rets[0].(string)
	if !ok {
		return addressResult{"FIRST_PARAM_NOT_STRING", ""}
	}
	return addressResult{SUCCESS, address}
}
