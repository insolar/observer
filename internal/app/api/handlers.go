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

type ServerInterfaceImpl struct {
}

func (s *ServerInterfaceImpl) Balance(w http.ResponseWriter, r *http.Request) {
}

func (s *ServerInterfaceImpl) Fee(w http.ResponseWriter, r *http.Request) {
}
func (s *ServerInterfaceImpl) Member(w http.ResponseWriter, r *http.Request) {
}
func (s *ServerInterfaceImpl) Notification(w http.ResponseWriter, r *http.Request) {
}
func (s *ServerInterfaceImpl) Transaction(w http.ResponseWriter, r *http.Request) {
}
func (s *ServerInterfaceImpl) TransactionList(w http.ResponseWriter, r *http.Request) {
}
func (s *ServerInterfaceImpl) TransactionsSearch(w http.ResponseWriter, r *http.Request) {
}
func (s *ServerInterfaceImpl) Coins(w http.ResponseWriter, r *http.Request) {
}
func (s *ServerInterfaceImpl) CoinsCirculating(w http.ResponseWriter, r *http.Request) {
}
func (s *ServerInterfaceImpl) CoinsMax(w http.ResponseWriter, r *http.Request) {
}
func (s *ServerInterfaceImpl) CoinsTotal(w http.ResponseWriter, r *http.Request) {
}
