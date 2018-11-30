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

package reply

import (
	"github.com/insolar/insolar/core"
)

// ErrHeavySyncInProgress returned when heavy sync in progress.
const (
	ErrHeavySyncInProgress ErrType = iota + 1
)

// HeavyError carries heavy record information
type HeavyError struct {
	Message string
	SubType ErrType
}

// Type implementation of Reply interface.
func (e *HeavyError) Type() core.ReplyType {
	return TypeHeavyError
}

// ConcreteType returns concrete error type.
func (e *HeavyError) ConcreteType() ErrType {
	return e.SubType
}

// Error returns error message for stored type.
func (e *HeavyError) Error() string {
	return e.Message
}