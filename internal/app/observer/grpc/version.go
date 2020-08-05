package grpc

import (
	"context"
	"strconv"

	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/mainnet/application/appfoundation"
	"google.golang.org/grpc/metadata"
)

func getCtxWithClientVersion(requestCtx context.Context) context.Context {
	ctxMD := metadata.AppendToOutgoingContext(requestCtx, exporter.KeyClientType, exporter.ValidateContractVersion.String())
	ctxMD = metadata.AppendToOutgoingContext(ctxMD, exporter.KeyClientVersionHeavy, strconv.Itoa(exporter.AllowedOnHeavyVersion))
	return metadata.AppendToOutgoingContext(ctxMD, exporter.KeyClientVersionContract, strconv.Itoa(appfoundation.AllowedVersionSmartContract))

}
