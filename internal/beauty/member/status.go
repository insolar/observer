package member

const (
	PENDING  = "PENDING"
	SUCCESS  = "SUCCESS"
	CANCELED = "CANCELED"
)

type memberResultParams struct {
	status           string
	migrationAddress string
	reference        string
}

func memberStatus(payload []byte) memberResultParams {
	rets := parsePayload(payload)
	if len(rets) < 2 {
		return memberResultParams{"NOT_ENOUGH_PAYLOAD_PARAMS", "", ""}
	}
	if retError, ok := rets[1].(error); ok {
		if retError != nil {
			return memberResultParams{CANCELED, "", ""}
		}
	}
	params, ok := rets[0].(map[string]interface{})
	if !ok {
		return memberResultParams{"FIRST_PARAM_NOT_MAP", "", ""}
	}
	referenceInterface, ok := params["reference"]
	if !ok {
		return memberResultParams{SUCCESS, "", ""}
	}
	reference, ok := referenceInterface.(string)
	if !ok {
		return memberResultParams{"MIGRATION_ADDRESS_NOT_STRING", "", ""}
	}

	migrationAddressInterface, ok := params["migrationAddress"]
	if !ok {
		return memberResultParams{SUCCESS, "", reference}
	}
	migrationAddress, ok := migrationAddressInterface.(string)
	if !ok {
		return memberResultParams{"MIGRATION_ADDRESS_NOT_STRING", "", reference}
	}
	return memberResultParams{SUCCESS, migrationAddress, reference}
}
