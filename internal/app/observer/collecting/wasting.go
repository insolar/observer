package collecting

import (
	"context"
	"fmt"

	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"

	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/store"
)

const (
	GetFreeMigrationAddress = "GetFreeMigrationAddress"
)

type VestingCollector struct {
	fetcher store.RecordFetcher
	log     insolar.Logger
}

func NewVestingCollector(log insolar.Logger, fetcher store.RecordFetcher) *VestingCollector {
	return &VestingCollector{
		log:     log,
		fetcher: fetcher,
	}
}

func (c *VestingCollector) Collect(ctx context.Context, rec *observer.Record) *observer.Vesting {
	if rec == nil {
		return nil
	}

	log := c.log.WithFields(
		map[string]interface{}{
			"collector": "VestingCollector",
			"record_id": rec.ID.DebugString(),
		})

	result, err := observer.CastToResult(rec)
	if err != nil {
		log.Warn(err.Error())
		return nil
	}

	if !result.IsSuccess(log) {
		return nil
	}

	requestID := result.Request()
	if requestID.IsEmpty() {
		panic(fmt.Sprintf("recordID %s: empty requestID from result", rec.ID.String()))
	}

	recordRequest, err := c.fetcher.Request(ctx, requestID)
	if err != nil {
		panic(errors.Wrapf(err, "recordID %s: failed to fetch request", rec.ID.String()))
	}

	request := recordRequest.Virtual.GetIncomingRequest()
	if request == nil {
		return nil
	}

	if request.Method != GetFreeMigrationAddress {
		return nil
	}

	address := ""
	result.ParseFirstPayloadValue(&address, log)
	return &observer.Vesting{
		Addr: address,
	}
}
