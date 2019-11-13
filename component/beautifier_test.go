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

package component

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/go-pg/pg"
	"github.com/insolar/insolar/application/api/requester"
	"github.com/insolar/insolar/application/builtin/contract/deposit"
	"github.com/insolar/insolar/application/builtin/proxy/migrationdaemon"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/insolar/insolar/ledger/heavy/exporter"
	"github.com/insolar/insolar/logicrunner/builtin/foundation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	proxyAccount "github.com/insolar/insolar/application/builtin/proxy/account"
	proxyCostCenter "github.com/insolar/insolar/application/builtin/proxy/costcenter"
	proxyDeposit "github.com/insolar/insolar/application/builtin/proxy/deposit"
	proxyDaemon "github.com/insolar/insolar/application/builtin/proxy/migrationdaemon"
	proxyWallet "github.com/insolar/insolar/application/builtin/proxy/wallet"

	"github.com/insolar/observer/configuration"
	"github.com/insolar/observer/internal/app/observer"
	"github.com/insolar/observer/internal/app/observer/collecting"
	"github.com/insolar/observer/internal/testutils"
	"github.com/insolar/observer/observability"
)

var db *pg.DB

type dbLogger struct{}

func (d dbLogger) BeforeQuery(q *pg.QueryEvent) {
	fmt.Println(q.FormattedQuery())
	return
}

func (d dbLogger) AfterQuery(q *pg.QueryEvent) {
	return
}

func TestMain(t *testing.M) {
	var dbCleaner func()
	db, _, dbCleaner = testutils.SetupDB("../scripts/migrations")

	// for debug purposes print all queries
	db.AddQueryHook(dbLogger{})

	retCode := t.Run()
	dbCleaner()
	os.Exit(retCode)
}

type fakeConn struct {
}

func (f fakeConn) PG() *pg.DB {
	return db
}

func TestBeautifier_Run(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		cfg := &configuration.Configuration{
			Replicator: configuration.Replicator{
				CacheSize: 100000,
			},
			LogLevel: "debug",
		}
		beautifier := makeBeautifier(cfg, observability.Make(cfg), fakeConn{})
		ctx := context.Background()

		beautifier(ctx, nil)
	})

	t.Run("happy path", func(t *testing.T) {
		cfg := &configuration.Configuration{
			Replicator: configuration.Replicator{
				CacheSize: 100000,
			},
			LogLevel: "debug",
		}
		beautifier := makeBeautifier(cfg, observability.Make(cfg), fakeConn{})
		ctx := context.Background()

		tdg := NewTreeDataGenerator()
		raw := &raw{
			batch: map[uint32]*exporter.Record{
				0: tdg.makeRequestWith("hello", gen.RecordReference(), nil),
			},
		}
		res := beautifier(ctx, raw)
		assert.NotNil(t, res)
	})

	t.Run("wastings", func(t *testing.T) {
		cfg := &configuration.Configuration{
			Replicator: configuration.Replicator{
				CacheSize: 100000,
			},
			LogLevel: "debug",
		}
		beautifier := makeBeautifier(cfg, observability.Make(cfg), fakeConn{})
		ctx := context.Background()

		tdg := NewTreeDataGenerator()

		pn := insolar.GenesisPulse.PulseNumber
		address := "0x5ca5e6417f818ba1c74d8f45104267a332c6aafb6ae446cc2bf8abd3735d1461111111111111111"
		out := tdg.makeOutgouingRequest(gen.Reference(), gen.Reference())
		call := tdg.makeGetMigrationAddressCall(pn)

		raw := &raw{
			batch: map[uint32]*exporter.Record{
				0: out,
				1: call,
				2: tdg.makeResultWith(out.Record.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
				3: tdg.makeResultWith(call.Record.ID, &foundation.Result{Returns: []interface{}{address, nil}}),
			},
		}
		res := beautifier(ctx, raw)
		assert.Equal(t, map[string]*observer.Wasting{
			address: {
				Addr: address,
			}}, res.wastings)
	})
}

func TestBeautifier_Deposit(t *testing.T) {
	cfg := &configuration.Configuration{
		Replicator: configuration.Replicator{
			CacheSize: 100000,
		},
		LogLevel: "debug",
	}
	beautifier := makeBeautifier(cfg, observability.Make(cfg), fakeConn{})
	ctx := context.Background()
	pn := insolar.GenesisPulse.PulseNumber
	tdg := NewTreeDataGenerator()

	call := tdg.makeDepositMigrationCall(pn)

	memberRef := gen.Reference()

	daemonCall := tdg.makeMigrationDaemonCall(pn, *insolar.NewReference(call.Record.ID))
	daemonCallIn := tdg.makeIncomingFromOutgoing(daemonCall.Record.Virtual.Union.(*record.Virtual_OutgoingRequest).OutgoingRequest)

	newDepositCall := tdg.makeNewDepositRequest(pn, *insolar.NewReference(daemonCallIn.Record.ID))
	newDepositCallIn := tdg.makeIncomingFromOutgoing(newDepositCall.Record.Virtual.Union.(*record.Virtual_OutgoingRequest).OutgoingRequest)

	balance := "123"
	amount := "456"
	txHash := "0x5ca5e6417f818ba1c74d8f45104267a332c6aafb6ae446cc2bf8abd3735d1461111111111111111"

	dep := deposit.Deposit{
		Balance:            balance,
		Amount:             amount,
		TxHash:             txHash,
		PulseDepositUnHold: pn + 10,
		Vesting:            10,
		VestingStep:        10,
	}
	memory, err := insolar.Serialize(dep)
	if err != nil {
		panic("fail serialize memory")
	}

	act := tdg.makeActivation(
		*insolar.NewReference(newDepositCallIn.Record.ID),
		*migrationdaemon.PrototypeReference,
		memory,
	)

	raw := &raw{
		batch: map[uint32]*exporter.Record{
			0: call,
			1: daemonCall,
			2: tdg.makeResultWith(daemonCall.Record.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
			3: daemonCallIn,
			4: tdg.makeResultWith(daemonCallIn.Record.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
			5: newDepositCall,
			6: tdg.makeResultWith(newDepositCall.Record.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
			7: newDepositCallIn,
			8: tdg.makeResultWith(newDepositCallIn.Record.ID, &foundation.Result{Returns: []interface{}{nil, nil}}),
			9: act,
			10: tdg.makeResultWith(call.Record.ID, &foundation.Result{Returns: []interface{}{
				migrationdaemon.DepositMigrationResult{Reference: memberRef.String()},
				nil,
			}}),
		},
	}
	transferDate, err := act.Record.ID.Pulse().AsApproximateTime()
	require.NoError(t, err)

	res := beautifier(ctx, raw)

	assert.Equal(t, 1, len(res.deposits))
	assert.Equal(t, map[insolar.ID]*observer.Deposit{
		act.Record.ID: {
			EthHash:         strings.ToLower(txHash),
			Ref:             *insolar.NewReference(newDepositCallIn.Record.ID),
			Member:          memberRef,
			Timestamp:       transferDate.Unix(),
			HoldReleaseDate: 1546300810,
			Amount:          amount,
			Balance:         balance,
			DepositState:    act.Record.ID,
			Vesting:         10,
			VestingStep:     10,
		},
	}, res.deposits)
}

type treeDataGenerator struct {
	Nonce uint64
}

func NewTreeDataGenerator() treeDataGenerator {
	return treeDataGenerator{Nonce: 0}
}

// not thread safe
func (t *treeDataGenerator) GetNonce() uint64 {
	nonce := t.Nonce
	t.Nonce++
	return nonce
}

func (t *treeDataGenerator) makeRequestWith(method string, reason insolar.Reference, args []byte) *exporter.Record {
	return &exporter.Record{
		Record: record.Material{
			ID: gen.ID(),
			Virtual: record.Virtual{Union: &record.Virtual_IncomingRequest{IncomingRequest: &record.IncomingRequest{
				Method:    method,
				Reason:    reason,
				Arguments: args,
				Nonce:     t.GetNonce(),
			}}}},
	}
}

// we need reasn for match too tree and some UNIQUE nonce
func (t *treeDataGenerator) makeOutgouingRequest(reason insolar.Reference, prototypeRef insolar.Reference) *exporter.Record {
	rec := &exporter.Record{
		Record: record.Material{
			ID: gen.ID(),
			Virtual: record.Virtual{
				Union: &record.Virtual_OutgoingRequest{
					OutgoingRequest: &record.OutgoingRequest{
						Reason:    reason,
						Prototype: &prototypeRef,
						Nonce:     t.GetNonce(),
					},
				},
			},
		},
	}
	return rec
}

// we need same nonce in makeOutgouingRequest and makeIncomingRequest
func (t *treeDataGenerator) makeIncomingFromOutgoing(outgoing *record.OutgoingRequest) *exporter.Record {
	rec := &exporter.Record{Record: record.Material{
		ID: gen.ID(),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Reason:    outgoing.Reason,
					Nonce:     outgoing.Nonce,
					Prototype: outgoing.Prototype,
					Method:    outgoing.Method,
					Arguments: outgoing.Arguments,
					Object:    outgoing.Object,
				},
			},
		},
	}}
	return rec
}

func (t *treeDataGenerator) makeActivation(ref insolar.Reference, prototypreRef insolar.Reference, memory []byte) *exporter.Record {
	rec := &exporter.Record{Record: record.Material{
		ID: gen.ID(),
		Virtual: record.Virtual{
			Union: &record.Virtual_Activate{
				Activate: &record.Activate{
					Request: ref,
					Image:   prototypreRef,
					Memory:  memory,
				},
			},
		},
	}}
	return rec
}

func (t *treeDataGenerator) makeGetMigrationAddressCall(pn insolar.PulseNumber) *exporter.Record {
	signature := ""
	pulseTimeStamp := 0
	raw, err := insolar.Serialize([]interface{}{nil, signature, pulseTimeStamp})
	if err != nil {
		panic("failed to serialize raw")
	}
	args, err := insolar.Serialize([]interface{}{raw})
	if err != nil {
		panic("failed to serialize arguments")
	}

	virtRecord := record.Wrap(&record.IncomingRequest{
		Method:    collecting.GetFreeMigrationAddress,
		Arguments: args,
	})

	rec := &exporter.Record{Record: record.Material{
		ID:      gen.IDWithPulse(pn),
		Virtual: virtRecord,
	}}
	return rec
}

func (t *treeDataGenerator) makeDepositMigrationCall(pn insolar.PulseNumber) *exporter.Record {
	request := &requester.ContractRequest{
		Params: requester.Params{
			CallSite:   collecting.CallSite,
			CallParams: nil,
		},
	}
	requestBody, err := json.Marshal(request)
	if err != nil {
		panic("failed to marshal request")
	}
	signature := ""
	pulseTimeStamp := 0
	raw, err := insolar.Serialize([]interface{}{requestBody, signature, pulseTimeStamp})
	if err != nil {
		panic("failed to serialize raw")
	}
	args, err := insolar.Serialize([]interface{}{raw})
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &exporter.Record{Record: record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Method:    "Call",
					Arguments: args,
					Nonce:     t.GetNonce(),
				},
			},
		},
	}}
	return rec
}

func (t *treeDataGenerator) makeMigrationDaemonCall(pn insolar.PulseNumber, reason insolar.Reference) *exporter.Record {
	signature := ""
	pulseTimeStamp := 0
	raw, err := insolar.Serialize([]interface{}{nil, signature, pulseTimeStamp})
	if err != nil {
		panic("failed to serialize raw")
	}
	args, err := insolar.Serialize([]interface{}{raw})
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &exporter.Record{Record: record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_OutgoingRequest{
				OutgoingRequest: &record.OutgoingRequest{
					Nonce:     t.GetNonce(),
					Method:    "DepositMigrationCall",
					Arguments: args,
					Prototype: proxyDaemon.PrototypeReference,
					Reason:    reason,
				},
			},
		},
	}}
	return rec
}

func (t *treeDataGenerator) makeNewDepositRequest(pn insolar.PulseNumber, reason insolar.Reference) *exporter.Record {
	signature := ""
	pulseTimeStamp := 0
	raw, err := insolar.Serialize([]interface{}{nil, signature, pulseTimeStamp})
	if err != nil {
		panic("failed to serialize raw")
	}
	args, err := insolar.Serialize([]interface{}{raw})
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &exporter.Record{Record: record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_OutgoingRequest{
				OutgoingRequest: &record.OutgoingRequest{
					Nonce:     t.GetNonce(),
					Method:    "New",
					Arguments: args,
					Prototype: proxyDeposit.PrototypeReference,
					Reason:    reason,
				},
			},
		},
	}}
	return rec
}

func (t *treeDataGenerator) makeResultWith(requestID insolar.ID, result *foundation.Result) *exporter.Record {
	payload, err := insolar.Serialize(result)
	if err != nil {
		panic("failed to serialize result")
	}
	ref := insolar.NewReference(requestID)
	rec := &exporter.Record{Record: record.Material{
		ID: gen.ID(),
		Virtual: record.Virtual{
			Union: &record.Virtual_Result{
				Result: &record.Result{
					Request: *ref,
					Payload: payload,
				},
			},
		},
	}}
	return rec
}

func (t *treeDataGenerator) makeWalletTransferCall(pn insolar.PulseNumber, reason insolar.Reference) *exporter.Record {
	signature := ""
	pulseTimeStamp := 0
	raw, err := insolar.Serialize([]interface{}{nil, signature, pulseTimeStamp})
	if err != nil {
		panic("failed to serialize raw")
	}
	args, err := insolar.Serialize([]interface{}{raw})
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &exporter.Record{Record: record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_OutgoingRequest{
				OutgoingRequest: &record.OutgoingRequest{
					Nonce:     t.GetNonce(),
					Method:    "Transfer",
					Arguments: args,
					Prototype: proxyWallet.PrototypeReference,
					Reason:    reason,
				},
			},
		},
	}}
	return rec
}

func (t *treeDataGenerator) makeAccountTransferCall(pn insolar.PulseNumber, reason insolar.Reference) *exporter.Record {
	signature := ""
	pulseTimeStamp := 0
	raw, err := insolar.Serialize([]interface{}{nil, signature, pulseTimeStamp})
	if err != nil {
		panic("failed to serialize raw")
	}
	args, err := insolar.Serialize([]interface{}{raw})
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &exporter.Record{Record: record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_OutgoingRequest{
				OutgoingRequest: &record.OutgoingRequest{
					Nonce:     t.GetNonce(),
					Method:    "Transfer",
					Arguments: args,
					Prototype: proxyAccount.PrototypeReference,
					Reason:    reason,
				},
			},
		},
	}}
	return rec
}

func (t *treeDataGenerator) makeCalcFeeCall(pn insolar.PulseNumber, reason insolar.Reference) *exporter.Record {
	signature := ""
	pulseTimeStamp := 0
	raw, err := insolar.Serialize([]interface{}{nil, signature, pulseTimeStamp})
	if err != nil {
		panic("failed to serialize raw")
	}
	args, err := insolar.Serialize([]interface{}{raw})
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &exporter.Record{Record: record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_OutgoingRequest{
				OutgoingRequest: &record.OutgoingRequest{
					Nonce:     t.GetNonce(),
					Method:    "CalcFee",
					Arguments: args,
					Prototype: proxyCostCenter.PrototypeReference,
					Reason:    reason,
				},
			},
		},
	}}
	return rec
}

func (t *treeDataGenerator) makeGetFeeMemberCall(pn insolar.PulseNumber, reason insolar.Reference) *exporter.Record {
	signature := ""
	pulseTimeStamp := 0
	raw, err := insolar.Serialize([]interface{}{nil, signature, pulseTimeStamp})
	if err != nil {
		panic("failed to serialize raw")
	}
	args, err := insolar.Serialize([]interface{}{raw})
	if err != nil {
		panic("failed to serialize arguments")
	}
	rec := &exporter.Record{Record: record.Material{
		ID: gen.IDWithPulse(pn),
		Virtual: record.Virtual{
			Union: &record.Virtual_OutgoingRequest{
				OutgoingRequest: &record.OutgoingRequest{
					Nonce:     t.GetNonce(),
					Method:    "GetFeeMember",
					Arguments: args,
					Prototype: proxyCostCenter.PrototypeReference,
					Reason:    reason,
				},
			},
		},
	}}
	return rec
}
