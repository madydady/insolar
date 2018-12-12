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

package core

import (
	"bytes"
	"encoding/binary"
	"encoding/json"

	"github.com/jbenet/go-base58"
)

const (
	// RecordHashSize is a record hash size. We use 224-bit SHA-3 hash (28 bytes).
	RecordHashSize = 28
	// RecordIDSize is relative record address.
	RecordIDSize = PulseNumberSize + RecordHashSize
	// RecordRefSize is absolute records address (including domain ID).
	RecordRefSize = RecordIDSize * 2
)

// RecordID is a unified record ID.
type RecordID [RecordIDSize]byte

// String implements stringer on RecordID and returns base58 encoded value
func (id *RecordID) String() string {
	return base58.Encode(id[:])
}

// NewRecordID generates RecordID byte representation.
func NewRecordID(pulse PulseNumber, hash []byte) *RecordID {
	var id RecordID
	copy(id[:PulseNumberSize], pulse.Bytes())
	copy(id[PulseNumberSize:], hash)
	return &id
}

// Bytes returns byte slice of RecordID.
func (id *RecordID) Bytes() []byte {
	return id[:]
}

// Pulse returns a copy of Pulse part of RecordID.
func (id *RecordID) Pulse() PulseNumber {
	pulse := binary.BigEndian.Uint32(id[:PulseNumberSize])
	return PulseNumber(pulse)
}

// Hash returns a copy of Hash part of RecordID.
func (id *RecordID) Hash() []byte {
	recHash := make([]byte, RecordHashSize)
	copy(recHash, id[PulseNumberSize:])
	return recHash
}

// Equal checks if reference points to the same record.
func (id *RecordID) Equal(other *RecordID) bool {
	if id == nil || other == nil {
		return false
	}
	return *id == *other
}

// MarshalJSON serializes ID into JSON.
func (id *RecordID) MarshalJSON() ([]byte, error) {
	if id == nil {
		return json.Marshal(nil)
	}
	return json.Marshal(base58.Encode(id[:]))
}

// RecordRef is a unified record reference.
type RecordRef [RecordRefSize]byte

// NewRecordRef returns RecordRef composed from domain and record
func NewRecordRef(domain RecordID, record RecordID) *RecordRef {
	var ref RecordRef
	ref.SetDomain(domain)
	ref.SetRecord(record)
	return &ref
}

// SetDomain set domain's RecordID.
func (ref *RecordRef) SetDomain(recID RecordID) {
	copy(ref[RecordIDSize:], recID[:])
}

// SetRecord set record's RecordID.
func (ref *RecordRef) SetRecord(recID RecordID) {
	copy(ref[:RecordIDSize], recID[:])
}

// Domain returns domain ID part of reference.
func (ref RecordRef) Domain() *RecordID {
	var id RecordID
	copy(id[:], ref[RecordIDSize:])
	return &id
}

// Record returns record's RecordID.
func (ref *RecordRef) Record() *RecordID {
	if ref == nil {
		return nil
	}
	var id RecordID
	copy(id[:], ref[:RecordIDSize])
	return &id
}

// String outputs base58 RecordRef representation.
func (ref RecordRef) String() string {
	return base58.Encode(ref[:])
}

// FromSlice : After CBOR Marshal/Unmarshal Ref can be converted to byte slice, this converts it back
func (ref RecordRef) FromSlice(from []byte) RecordRef {
	for i := 0; i < RecordRefSize; i++ {
		ref[i] = from[i]
	}
	return ref
}

// Bytes returns byte slice of RecordRef.
func (ref RecordRef) Bytes() []byte {
	return ref[:]
}

// Equal checks if reference points to the same record.
func (ref RecordRef) Equal(other RecordRef) bool {
	return ref == other
}

// IsEmpty - check for void
func (ref RecordRef) IsEmpty() bool {
	return ref.Equal(RecordRef{})
}

// Compare compares two record references
func (ref RecordRef) Compare(other RecordRef) int {
	return bytes.Compare(ref.Bytes(), other.Bytes())
}

// NewRefFromBase58 deserializes reference from base58 encoded string.
func NewRefFromBase58(str string) RecordRef {
	// TODO: if str < 20 bytes, always returns 0. need to check this.
	decoded := base58.Decode(str)
	var ref RecordRef
	copy(ref[:], decoded)
	return ref
}

// MarshalJSON serializes reference into JSON.
func (ref *RecordRef) MarshalJSON() ([]byte, error) {
	if ref == nil {
		return json.Marshal(nil)
	}
	rec, err := ref.Record().MarshalJSON()
	if err != nil {
		return nil, err
	}
	domain, err := ref.Domain().MarshalJSON()
	if err != nil {
		return nil, err
	}
	return json.Marshal(bytes.Join([][]byte{rec, domain}, nil))
}
