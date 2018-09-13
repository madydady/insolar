/*
 *    Copyright 2018 Insolar
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package experiment

import (
	"testing"

	"github.com/insolar/insolar/genesis/experiment/member"
	"github.com/insolar/insolar/genesis/experiment/wallet"
	"github.com/insolar/insolar/toolkit/go/foundation"
	"github.com/stretchr/testify/assert"
)

func TestTransferViaReceive(t *testing.T) {
	// Create member which balance will increase
	toMember := member.NewMember("Vasya")
	toMemberRef := foundation.SaveToLedger(toMember, member.TypeReference)

	// Create member which balance will decrease
	fromMember := member.NewMember("Petya")
	fromMemberRef := foundation.SaveToLedger(fromMember, member.TypeReference)

	// Create wallet for toMember
	toWallet := wallet.NewWallet(1000)

	// Create wallet for fromMember
	fromWallet := wallet.NewWallet(2000)

	// Make fromWallet delegate of fromMember
	fromMember.InjectDelegate(fromWallet, wallet.TypeReference)
	// Make toWallet delegate of toMember
	toMember.InjectDelegate(toWallet, wallet.TypeReference)

	// Get fromMember as wallet instance
	fromMemberAsWallet, ok := foundation.GetImplementationFor(fromMemberRef, wallet.TypeReference).(*wallet.Wallet)
	assert.True(t, ok)
	assert.NotNil(t, fromMemberAsWallet)
	assert.Equal(t, fromWallet.GetReference(), fromMemberAsWallet.GetReference())

	// Get toMember as wallet instance
	toMemberAsWallet, ok := foundation.GetImplementationFor(toMemberRef, wallet.TypeReference).(*wallet.Wallet)
	assert.True(t, ok)
	assert.Equal(t, toWallet, toMemberAsWallet)
	assert.Equal(t, toWallet.GetReference(), toMemberAsWallet.GetReference())

	// Inject fake context of Caller for test
	foundation.InjectFakeContext(3, &foundation.CallContext{Caller: toWallet.GetReference()})

	// Call to get money from one member to another
	toMemberAsWallet.Receive(500, fromMemberRef)

	// Check balance
	assert.Equal(t, 1500, int(fromWallet.GetTotalBalance()))
	assert.Equal(t, 1500, int(toWallet.GetTotalBalance()))
}

func TestTransferViaTransfer(t *testing.T) {
	// Create member which balance will increase
	toMember := member.NewMember("Vasya")
	toMemberRef := foundation.SaveToLedger(toMember, member.TypeReference)

	// Create member which balance will decrease
	fromMember := member.NewMember("Petya")
	fromMemberRef := foundation.SaveToLedger(fromMember, member.TypeReference)

	// Create wallet for toMember
	toWallet := wallet.NewWallet(1000)

	// Create wallet for fromMember
	fromWallet := wallet.NewWallet(2000)

	// Make fromWallet delegate of fromMember
	fromMember.InjectDelegate(fromWallet, wallet.TypeReference)
	// Make toWallet delegate of toMember
	toMember.InjectDelegate(toWallet, wallet.TypeReference)

	// Get fromMember as wallet instance
	fromMemberAsWallet, ok := foundation.GetImplementationFor(fromMemberRef, wallet.TypeReference).(*wallet.Wallet)
	assert.True(t, ok)
	assert.NotNil(t, fromMemberAsWallet)
	assert.Equal(t, fromWallet.GetReference(), fromMemberAsWallet.GetReference())

	// Get toMember as wallet instance
	toMemberAsWallet, ok := foundation.GetImplementationFor(toMemberRef, wallet.TypeReference).(*wallet.Wallet)
	assert.True(t, ok)
	assert.Equal(t, toWallet, toMemberAsWallet)
	assert.Equal(t, toWallet.GetReference(), toMemberAsWallet.GetReference())

	// Inject fake context of Caller for test
	foundation.InjectFakeContext(2, &foundation.CallContext{Caller: toWallet.GetReference()})

	// Call to get money from one member to another
	fromMemberAsWallet.Transfer(500, toMemberRef)

	// Check balance
	assert.Equal(t, 1500, int(fromWallet.GetTotalBalance()))
	assert.Equal(t, 1500, int(toWallet.GetTotalBalance()))
}