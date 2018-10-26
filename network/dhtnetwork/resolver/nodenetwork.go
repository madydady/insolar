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

package resolver

import (
	"github.com/insolar/insolar/core"
	"github.com/jbenet/go-base58"
	"golang.org/x/crypto/sha3"
)

// ResolveHostID returns a host found by reference.
func ResolveHostID(ref core.RecordRef) string {
	hash := make([]byte, 20)
	sha3.ShakeSum128(hash, ref[:])
	return base58.Encode(hash)
}