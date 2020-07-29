package grpc

import (
	"context"
	"strconv"

	"github.com/insolar/insolar/ledger/heavy/exporter"
	"google.golang.org/grpc/metadata"
)

func getCtxWithClientVersion(requestCtx context.Context) context.Context {
	ctxMD := metadata.AppendToOutgoingContext(requestCtx, exporter.KeyClientType, exporter.ValidateContractVersion.String())
	ctxMD = metadata.AppendToOutgoingContext(ctxMD, exporter.KeyClientVersionHeavy, strconv.Itoa(exporter.AllowedOnHeavyVersion))
	// TODO set real version contract https://insolar.atlassian.net/browse/MN-631
	return metadata.AppendToOutgoingContext(ctxMD, exporter.KeyClientVersionContract, "2")

}
