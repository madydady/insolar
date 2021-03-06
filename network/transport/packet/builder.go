/*
 * The Clear BSD License
 *
 * Copyright (c) 2019 Insolar Technologies
 *
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without modification, are permitted (subject to the limitations in the disclaimer below) provided that the following conditions are met:
 *
 *  Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
 *  Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
 *  Neither the name of Insolar Technologies nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.
 *
 * NO EXPRESS OR IMPLIED LICENSES TO ANY PARTY'S PATENT RIGHTS ARE GRANTED BY THIS LICENSE. THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 *
 */

package packet

import (
	"github.com/insolar/insolar/network"
	"github.com/insolar/insolar/network/transport/host"
	"github.com/insolar/insolar/network/transport/packet/types"
)

// Builder allows lazy building of packets.
// Each operation returns new copy of a builder.
type Builder struct {
	actions []func(packet *Packet)
}

// NewBuilder returns empty packet builder.
func NewBuilder(sender *host.Host) Builder {
	cb := Builder{}
	cb.actions = append(cb.actions, func(packet *Packet) {
		packet.Sender = sender
		packet.RemoteAddress = sender.Address.String()
	})
	return cb
}

// Build returns configured packet.
func (cb Builder) Build() (packet *Packet) {
	packet = &Packet{}
	for _, action := range cb.actions {
		action(packet)
	}
	return
}

// Receiver sets packet receiver.
func (cb Builder) Receiver(host *host.Host) Builder {
	cb.actions = append(cb.actions, func(packet *Packet) {
		packet.Receiver = host
	})
	return cb
}

// Type sets packet type.
func (cb Builder) Type(packetType types.PacketType) Builder {
	cb.actions = append(cb.actions, func(packet *Packet) {
		packet.Type = packetType
	})
	return cb
}

// Request adds request data to packet.
func (cb Builder) Request(request interface{}) Builder {
	cb.actions = append(cb.actions, func(packet *Packet) {
		packet.Data = request
	})
	return cb
}

func (cb Builder) RequestID(id network.RequestID) Builder {
	cb.actions = append(cb.actions, func(packet *Packet) {
		packet.RequestID = id
	})
	return cb
}

func (cb Builder) TraceID(traceID string) Builder {
	cb.actions = append(cb.actions, func(packet *Packet) {
		packet.TraceID = traceID
	})
	return cb
}

// Response adds response data to packet
func (cb Builder) Response(response interface{}) Builder {
	cb.actions = append(cb.actions, func(packet *Packet) {
		packet.Data = response
		packet.IsResponse = true
	})
	return cb
}

// Error adds error description to packet.
func (cb Builder) Error(err error) Builder {
	cb.actions = append(cb.actions, func(packet *Packet) {
		packet.Error = err
	})
	return cb
}
