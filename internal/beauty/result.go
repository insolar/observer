package beauty

import (
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/record"
)

type Result struct {
	tableName struct{} `sql:"results"`

	ResultID string `sql:",pk,column_name:result_id"`
	Request  string
	Payload  string

	requestID insolar.ID
}

func (b *Beautifier) parseResult(id insolar.ID, res *record.Result) {
	b.rawResults[id] = &Result{
		ResultID: id.String(),
		Request:  res.Request.String(),
		//Payload:  string(res.Payload),
	}
}

func (b *Beautifier) storeResult(result *Result) error {
	_, err := b.db.Model(result).OnConflict("DO NOTHING").Insert()
	if err != nil {
		return err
	}
	return nil
}
