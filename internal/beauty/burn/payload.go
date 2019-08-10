package burn

import (
	"github.com/insolar/insolar/insolar"
	log "github.com/sirupsen/logrus"
)

func parsePayload(payload []byte) []interface{} {
	rets := []interface{}{}
	err := insolar.Deserialize(payload, &rets)
	if err != nil {
		log.Warnf("failed to parse payload as two interfaces")
		return []interface{}{}
	}
	return rets
}
