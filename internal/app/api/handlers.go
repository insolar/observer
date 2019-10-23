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

package api

import (
	"net/http"
)

type ObserverApi struct {
}

func (s *ObserverApi) Balance(w http.ResponseWriter, r *http.Request) {
}

func (s *ObserverApi) Fee(w http.ResponseWriter, r *http.Request) {
}
func (s *ObserverApi) Member(w http.ResponseWriter, r *http.Request) {
}
func (s *ObserverApi) Notification(w http.ResponseWriter, r *http.Request) {
}
func (s *ObserverApi) Transaction(w http.ResponseWriter, r *http.Request) {
}
func (s *ObserverApi) TransactionList(w http.ResponseWriter, r *http.Request) {
}
func (s *ObserverApi) TransactionsSearch(w http.ResponseWriter, r *http.Request) {
}
func (s *ObserverApi) Coins(w http.ResponseWriter, r *http.Request) {
}
func (s *ObserverApi) CoinsCirculating(w http.ResponseWriter, r *http.Request) {
}
func (s *ObserverApi) CoinsMax(w http.ResponseWriter, r *http.Request) {
}
func (s *ObserverApi) CoinsTotal(w http.ResponseWriter, r *http.Request) {
}
