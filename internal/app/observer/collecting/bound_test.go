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

package collecting

import (
	"testing"

	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/gen"
	"github.com/insolar/insolar/insolar/record"
	"github.com/stretchr/testify/require"

	"github.com/insolar/observer/internal/app/observer"
)

type mockedResultCollector struct {
	collectFunc func(*observer.Record) *observer.CoupledResult
}

func (m *mockedResultCollector) Collect(r *observer.Record) *observer.CoupledResult {
	return m.collectFunc(r)
}

type mockedActivateCollector struct {
	collectFunc func(*observer.Record) *observer.CoupledActivate
}

func (m *mockedActivateCollector) Collect(r *observer.Record) *observer.CoupledActivate {
	return m.collectFunc(r)
}

func TestBoundCollector_NilCollector(t *testing.T) {
	collector := NewBoundCollector(nil, nil)
	require.Nil(t, collector.Collect(nil))
}

func TestBoundCollector_InnerNilCollect(t *testing.T) {
	collector := NewBoundCollector(&mockedResultCollector{
		collectFunc: func(*observer.Record) *observer.CoupledResult {
			return nil
		},
	}, &mockedActivateCollector{
		collectFunc: func(*observer.Record) *observer.CoupledActivate {
			return nil
		},
	})

	require.Nil(t, collector.Collect(&observer.Record{}))
}

func makeRequest(ID insolar.ID, method string, prototype *insolar.Reference, args []byte) *observer.Request {
	return &observer.Request{
		ID: ID,
		Virtual: record.Virtual{Union: &record.Virtual_IncomingRequest{IncomingRequest: &record.IncomingRequest{
			Method:    method,
			Prototype: prototype,
			Arguments: args,
		}}}}
}

func makeRequestWithReason(ID insolar.ID, Reason insolar.Reference, method string, prototype *insolar.Reference, args []byte) *observer.Request {
	return &observer.Request{
		ID: ID,
		Virtual: record.Virtual{Union: &record.Virtual_IncomingRequest{IncomingRequest: &record.IncomingRequest{
			Reason:    Reason,
			Method:    method,
			Prototype: prototype,
			Arguments: args,
		}}}}
}

func TestNewBoundCollector_Couple(t *testing.T) {
	requestRef := gen.RecordReference()
	protoRef := gen.RecordReference()

	resultList := []*observer.CoupledResult{
		{
			Request: makeRequest(*requestRef.GetLocal(), "some", &protoRef, nil),
			Result: &observer.Result{
				ID: gen.ID(),
			},
		},
	}

	activeList := []*observer.CoupledActivate{
		{
			Request: makeRequestWithReason(gen.ID(), requestRef, "some", &protoRef, nil),
			Activate: &observer.Activate{
				ID: gen.ID(),
			},
		},
	}

	var expected []*BoundCouple

	for i := range resultList {
		expected = append(expected, &BoundCouple{
			Result:   resultList[i].Result,
			Activate: activeList[i].Activate,
		})
	}

	collector := NewBoundCollector(&mockedResultCollector{
		collectFunc: func(*observer.Record) *observer.CoupledResult {
			var res *observer.CoupledResult
			if len(resultList) == 0 {
				return res
			}
			res, resultList = resultList[len(resultList)-1], resultList[:len(resultList)-1]
			return res
		},
	}, &mockedActivateCollector{
		collectFunc: func(*observer.Record) *observer.CoupledActivate {
			var res *observer.CoupledActivate
			if len(activeList) == 0 {
				return res
			}
			res, activeList = activeList[len(activeList)-1], activeList[:len(activeList)-1]
			return res
		},
	})

	records := []*observer.Record{
		{}, {},
	}
	var actual []*BoundCouple
	for _, r := range records {
		deposit := collector.Collect(r)
		if deposit != nil {
			actual = append(actual, deposit)
		}
	}

	require.Len(t, actual, 1)
	require.Equal(t, expected, actual)
}
