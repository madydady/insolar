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

package servicenetwork

import (
	"context"
	"time"

	"github.com/insolar/insolar/component"
	"github.com/insolar/insolar/configuration"
	"github.com/insolar/insolar/core"
	"github.com/insolar/insolar/network/pulsenetwork"
	"github.com/insolar/insolar/network/transport"
	"github.com/insolar/insolar/network/transport/relay"
	"github.com/insolar/insolar/pulsar/entropygenerator"
	"github.com/pkg/errors"
)

type TestPulsar interface {
	Start(ctx context.Context, bootstrapHosts []string) error
	component.Stopper
}

func NewTestPulsar(pulseTimeMs, requestsTimeoutMs, pulseDelta int32) (TestPulsar, error) {
	transportCfg := configuration.Transport{
		Protocol:  "TCP",
		Address:   "127.0.0.1:0",
		BehindNAT: false,
	}
	tp, err := transport.NewTransport(transportCfg, relay.NewProxy())
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create distributor transport")
	}
	return &testPulsar{
		transport:         tp,
		generator:         &entropygenerator.StandardEntropyGenerator{},
		pulseTimeMs:       pulseTimeMs,
		reqTimeoutMs:      requestsTimeoutMs,
		pulseDelta:        pulseDelta,
		cancellationToken: make(chan struct{}),
	}, nil
}

type testPulsar struct {
	transport   transport.Transport
	distributor core.PulseDistributor
	generator   entropygenerator.EntropyGenerator
	cm          *component.Manager

	pulseTimeMs  int32
	reqTimeoutMs int32
	pulseDelta   int32

	cancellationToken chan struct{}
}

func (tp *testPulsar) Start(ctx context.Context, bootstrapHosts []string) error {
	var err error
	distributorCfg := configuration.PulseDistributor{
		BootstrapHosts:            bootstrapHosts,
		PingRequestTimeout:        tp.reqTimeoutMs,
		RandomHostsRequestTimeout: tp.reqTimeoutMs,
		PulseRequestTimeout:       tp.reqTimeoutMs,
		RandomNodesCount:          1,
	}
	tp.distributor, err = pulsenetwork.NewDistributor(distributorCfg)
	if err != nil {
		return errors.Wrap(err, "Failed to create pulse distributor")
	}

	tp.cm = &component.Manager{}
	tp.cm.Inject(tp.transport, tp.distributor)

	if err = tp.cm.Init(ctx); err != nil {
		return errors.Wrap(err, "Failed to init test pulsar components")
	}
	if err = tp.cm.Start(ctx); err != nil {
		return errors.Wrap(err, "Failed to start test pulsar components")
	}

	go tp.distribute(ctx)
	return nil
}

func (tp *testPulsar) distribute(ctx context.Context) {
	timeNow := time.Now()
	pulseNumber := core.CalculatePulseNumber(timeNow)
	pulse := core.Pulse{
		PulseNumber:      pulseNumber,
		Entropy:          tp.generator.GenerateEntropy(),
		NextPulseNumber:  pulseNumber + core.PulseNumber(tp.pulseDelta),
		PrevPulseNumber:  pulseNumber - core.PulseNumber(tp.pulseDelta),
		EpochPulseNumber: 1,
		OriginID:         [16]byte{206, 41, 229, 190, 7, 240, 162, 155, 121, 245, 207, 56, 161, 67, 189, 0},
		PulseTimestamp:   timeNow.Unix(),
	}

	for {
		select {
		case <-time.After(time.Duration(tp.pulseTimeMs) * time.Millisecond):
			go tp.distributor.Distribute(ctx, pulse)
			pulse = tp.incrementPulse(pulse)
		case <-tp.cancellationToken:
			return
		}
	}
}

func (tp *testPulsar) incrementPulse(pulse core.Pulse) core.Pulse {
	newPulse := pulse.PulseNumber + core.PulseNumber(tp.pulseDelta)
	return core.Pulse{
		PulseNumber:      newPulse,
		Entropy:          tp.generator.GenerateEntropy(),
		NextPulseNumber:  newPulse + core.PulseNumber(tp.pulseDelta),
		PrevPulseNumber:  pulse.PulseNumber,
		EpochPulseNumber: pulse.EpochPulseNumber,
		OriginID:         pulse.OriginID,
		PulseTimestamp:   time.Now().Unix(),
	}
}

func (tp *testPulsar) Stop(ctx context.Context) error {
	if err := tp.cm.Stop(ctx); err != nil {
		return errors.Wrap(err, "Failed to stop test pulsar components")
	}
	close(tp.cancellationToken)
	return nil
}
