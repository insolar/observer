package transfer

const (
	PENDING  = "PENDING"
	SUCCESS  = "SUCCESS"
	CANCELED = "CANCELED"
)

type txResult struct {
	status string
	fee    string
}

func parseTransferResultPayload(payload []byte) txResult {
	rets := parsePayload(payload)
	if len(rets) < 2 {
		return txResult{status: "NOT_ENOUGH_PAYLOAD_PARAMS", fee: ""}
	}
	if retError, ok := rets[1].(error); ok {
		if retError != nil {
			return txResult{status: CANCELED, fee: ""}
		}
	}
	params, ok := rets[0].(map[string]interface{})
	if !ok {
		return txResult{status: "FIRST_PARAM_NOT_MAP", fee: ""}
	}
	feeInterface, ok := params["fee"]
	if !ok {
		return txResult{status: "FEE_PARAM_NOT_EXIST", fee: ""}
	}
	fee, ok := feeInterface.(string)
	if !ok {
		return txResult{status: "FEE_NOT_STRING", fee: ""}
	}
	return txResult{status: SUCCESS, fee: fee}
}
