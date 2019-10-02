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
	"reflect"

	"github.com/insolar/insolar/insolar"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/insolar/observer/internal/app/observer"
)

type BoundCollector struct {
	results   observer.ResultCollector
	activates observer.ActivateCollector
	chains    observer.ChainCollector
}

type BoundCouple struct {
	Result   *observer.Result
	Activate *observer.Activate
}

func NewBoundCollector(originRequest, properResult, directRequest, properActivate observer.Predicate) *BoundCollector {
	results := NewResultCollector(originRequest, properResult)
	activates := NewActivateCollector(directRequest, properActivate)
	parent := &RelationDesc{
		Is:     isCoupledResult,
		Origin: coupledResultOrigin,
		Proper: isCoupledResult,
	}
	child := &RelationDesc{
		Is:     isCoupledActivate,
		Origin: coupledActivateOrigin,
		Proper: isCoupledActivate,
	}
	return &BoundCollector{
		results:   results,
		activates: activates,
		chains:    NewChainCollector(parent, child),
	}
}

func (c *BoundCollector) Collect(rec *observer.Record) *BoundCouple {
	if rec == nil {
		return nil
	}
	res := c.results.Collect(rec)
	act := c.activates.Collect(rec)

	var (
		fullChain *observer.Chain
	)
	switch {
	case act != nil:
		fullChain = c.chains.Collect(act)
	case res != nil:
		fullChain = c.chains.Collect(res)
	}

	if fullChain == nil {
		return nil
	}
	actRecord, resRecord := unwrapChain(fullChain)
	return &BoundCouple{
		Result:   resRecord,
		Activate: actRecord,
	}
}

func isCoupledResult(chain interface{}) bool {
	_, ok := chain.(*observer.CoupledResult)
	return ok
}

func isCoupledActivate(chain interface{}) bool {
	_, ok := chain.(*observer.CoupledActivate)
	return ok
}

func coupledResultOrigin(chain interface{}) insolar.ID {
	coupled, ok := chain.(*observer.CoupledResult)
	if !ok {
		return insolar.ID{}
	}
	request := coupled.Request
	if !request.IsIncoming() {
		log.Warnf("failed to use not incoming request to get origin")
	}
	return request.ID
}

func coupledActivateOrigin(chain interface{}) insolar.ID {
	coupled, ok := chain.(*observer.CoupledActivate)
	if !ok {
		return insolar.ID{}
	}
	request := coupled.Request
	if !request.IsIncoming() {
		log.Warnf("failed to use not incoming request to get origin")
	}
	return request.Reason()
}

func unwrapChain(chain *observer.Chain) (*observer.Activate, *observer.Result) {
	coupledAct, ok := chain.Child.(*observer.CoupledActivate)
	if !ok {
		log.Error(errors.Errorf("trying to use %s as *observer.Chain", reflect.TypeOf(chain.Child)))
		return nil, nil
	}
	if coupledAct.Activate == nil {
		log.Error(errors.New("invalid coupled activate chain, child is nil"))
		return nil, nil
	}
	actRecord := coupledAct.Activate

	coupledRes, ok := chain.Parent.(*observer.CoupledResult)
	if !ok {
		log.Error(errors.Errorf("trying to use %s as *observer.Chain", reflect.TypeOf(chain.Parent)))
		return nil, nil
	}
	if coupledRes.Result == nil {
		log.Error(errors.New("invalid coupled result chain, child is nil"))
		return nil, nil
	}
	resRecord := coupledRes.Result
	return actRecord, resRecord
}
