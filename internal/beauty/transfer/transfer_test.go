//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package transfer

import (
	"context"
	"encoding/hex"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	memberProxy "github.com/insolar/insolar/logicrunner/builtin/proxy/member"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/db"
	"github.com/insolar/observer/internal/dto"
	"github.com/insolar/observer/internal/metrics"
	"github.com/insolar/observer/internal/model/beauty"
	"github.com/insolar/observer/internal/replicator"
)

func makeOutgouingRequest() *record.Material {
	return &record.Material{
		ID: gen.ID(),
		Virtual: record.Virtual{
			Union: &record.Virtual_OutgoingRequest{
				OutgoingRequest: &record.OutgoingRequest{},
			},
		},
	}
}

func makeResultWith(requestID insolar.ID) *record.Material {
	ref := insolar.NewReference(requestID)
	return &record.Material{
		ID: gen.ID(),
		Virtual: record.Virtual{
			Union: &record.Virtual_Result{
				Result: &record.Result{
					Request: *ref,
				},
			},
		},
	}
}

func makeCallRequest() *record.Material {
	return &record.Material{
		ID: gen.ID(),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Method:    "Call",
					Prototype: memberProxy.PrototypeReference,
				},
			},
		},
	}
}

func makeTransfer() []*record.Material {
	return []*record.Material{makeCallRequest()}
}

func loadSample(path string) *record.Material {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		panic(errors.New("failed to read transfer call record from test file"))
	}
	transferCallSample := strings.TrimRight(string(content), "\n")
	data, err := hex.DecodeString(transferCallSample)
	if err != nil {
		panic(errors.New("failed to decode serialized record"))
	}
	r := &record.Material{}
	err = r.Unmarshal(data)
	if err != nil {
		panic(errors.New("failed to parse serialized record"))
	}
	return r
}

func makeTransactionMock() *pg.Tx {
	return &pg.Tx{}
}

func makeSampleTransfer(reverse bool) []*record.Material {
	out := makeOutgouingRequest()
	outRes := makeResultWith(out.ID)
	req := loadSample("./testdata/transfer_call_request.hex")
	res := loadSample("./testdata/transfer_call_result.hex")
	if reverse {
		return []*record.Material{res, req, outRes, out}
	}
	return []*record.Material{req, res, out, outRes}
}

func TestNewComposer(t *testing.T) {
	require.NotNil(t, NewComposer())
}

func Test_parseTransferCallParams(t *testing.T) {
	req := loadSample("./testdata/transfer_call_request.hex")
	expected := &transferCallParams{
		Amount:            "18",
		ToMemberReference: "5fcCG3KwMFPYKpNC3AwBcEU2JeYQk8kQ9cPtqTmBvoW.11111111111111111111111111111111",
	}
	request := (*dto.Request)(req)
	actual := &transferCallParams{}
	request.ParseMemberContractCallParams(actual)
	require.Equal(t, expected, actual)
	logrus.Infof("req %v", req.ID.Pulse())

	expectedResult := &transferResult{
		Fee: "8",
	}
	res := loadSample("./testdata/transfer_call_result.hex")
	result := (*dto.Result)(res)
	actualResult := &transferResult{}
	result.ParseFirstPayloadValue(actualResult)
	require.Equal(t, expectedResult, actualResult)
	logrus.Infof("res %v", res)
}

func TestComposer_ProcessDump(t *testing.T) {
	t.Run("check_sample", func(t *testing.T) {
		sampleTransfer := &beauty.Transfer{
			TxID:          "5fnpnGwtS1wRzU466CASNjhgeY1GvZJZnasdL6C5GW1.11111111111111111111111111111111",
			Amount:        "18",
			Fee:           "8",
			TransferDate:  1566301291,
			PulseNum:      20066028,
			Status:        "SUCCESS",
			MemberFromRef: "5fnh42GBdHUfb69z1wgda6qngZ3rFkonjY85cQWJr8d.11111111111111111111111111111111",
			MemberToRef:   "5fcCG3KwMFPYKpNC3AwBcEU2JeYQk8kQ9cPtqTmBvoW.11111111111111111111111111111111",
			WalletFromRef: "TODO",
			WalletToRef:   "TODO",
			EthHash:       "",
		}
		composer := NewComposer()

		transfer := makeSampleTransfer(false)
		for _, rec := range transfer {
			composer.Process(rec)
		}

		require.Len(t, composer.cache, 1)
		require.Equal(t, sampleTransfer, composer.cache[0])

		tx := db.NewDBMock(t)
		pub := replicator.NewOnDumpSuccessMock(t)
		pub.SubscribeMock.Set(func(h replicator.SuccessHandle) {
			tx.InsertMock.Set(func(model ...interface{}) error {
				require.Equal(t, sampleTransfer, model)
				h()
				return nil
			})
		})

		err := composer.Dump(tx, pub)
		require.NoError(t, err)
	})
	t.Run("check_sample_reverse", func(t *testing.T) {
		sampleTransfer := &beauty.Transfer{
			TxID:          "5fnpnGwtS1wRzU466CASNjhgeY1GvZJZnasdL6C5GW1.11111111111111111111111111111111",
			Amount:        "18",
			Fee:           "8",
			TransferDate:  1566301291,
			PulseNum:      20066028,
			Status:        "SUCCESS",
			MemberFromRef: "5fnh42GBdHUfb69z1wgda6qngZ3rFkonjY85cQWJr8d.11111111111111111111111111111111",
			MemberToRef:   "5fcCG3KwMFPYKpNC3AwBcEU2JeYQk8kQ9cPtqTmBvoW.11111111111111111111111111111111",
			WalletFromRef: "TODO",
			WalletToRef:   "TODO",
			EthHash:       "",
		}
		composer := NewComposer()

		transfer := makeSampleTransfer(true)
		for _, rec := range transfer {
			composer.Process(rec)
		}

		require.Len(t, composer.cache, 1)
		require.Equal(t, sampleTransfer, composer.cache[0])

		tx := db.NewDBMock(t)
		pub := replicator.NewOnDumpSuccessMock(t)
		pub.SubscribeMock.Set(func(h replicator.SuccessHandle) {
			tx.InsertMock.Set(func(model ...interface{}) error {
				require.Equal(t, sampleTransfer, model)
				h()
				return nil
			})
		})

		err := composer.Dump(tx, pub)
		require.NoError(t, err)
	})
}

func TestComposer_Init(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		ctx := context.Background()
		composer := NewComposer()
		require.NoError(t, composer.Init(ctx))
	})

	t.Run("with_registry", func(t *testing.T) {
		ctx := context.Background()
		composer := NewComposer()
		composer.Metrics = metrics.New()
		require.NoError(t, composer.Init(ctx))
	})
}
