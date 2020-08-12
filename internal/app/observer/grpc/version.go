package grpc

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/mainnet/application/appfoundation"
	"google.golang.org/grpc/metadata"
)

const delayForReadPrometheus = 12

const ClientDeprecated = 1

func getCtxWithClientVersion(requestCtx context.Context) context.Context {
	ctxMD := metadata.AppendToOutgoingContext(requestCtx, exporter.KeyClientType, exporter.ValidateContractVersion.String())
	ctxMD = metadata.AppendToOutgoingContext(ctxMD, exporter.KeyClientVersionHeavy, strconv.Itoa(exporter.AllowedOnHeavyVersion))
	return metadata.AppendToOutgoingContext(ctxMD, exporter.KeyClientVersionContract, strconv.Itoa(appfoundation.AllowedVersionSmartContract))

}

func detectedDeprecatedVersion(err error, logger insolar.Logger) {
	if !strings.Contains(err.Error(), exporter.ErrDeprecatedClientVersion.Error()) {
		return
	}
	isDeprecatedClient.Set(ClientDeprecated)
	time.Sleep(delayForReadPrometheus * time.Second)
	logger.Fatal(exporter.ErrDeprecatedClientVersion.Error())
}
