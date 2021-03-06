// +build networktest

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
	"fmt"
	"testing"
	"time"

	"github.com/insolar/insolar/consensus/phases"
	"github.com/insolar/insolar/core"
	"github.com/insolar/insolar/log"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	consensusMin    = 5 // minimum count of participants that can survive when one node leaves
	consensusMinMsg = fmt.Sprintf("skip test for bootstrap nodes < %d", consensusMin)
)

func (s *testSuite) TestNetworkConsensus3Times() {
	s.waitForConsensus(3)
}

func (s *testSuite) TestNodeConnect() {
	s.preInitNode(s.fixture().testNode)

	s.InitTestNode()
	s.StartTestNode()
	defer func() {
		s.StopTestNode()
	}()

	s.waitForConsensus(1)

	activeNodes := s.fixture().bootstrapNodes[0].serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(s.getNodesCount(), len(activeNodes))

	s.waitForConsensus(1)

	activeNodes = s.fixture().bootstrapNodes[0].serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(s.getNodesCount()+1, len(activeNodes))

	s.waitForConsensus(2)

	activeNodes = s.fixture().bootstrapNodes[0].serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(s.getNodesCount()+1, len(activeNodes))
}

func (s *testSuite) TestNodeLeave() {
	s.preInitNode(s.fixture().testNode)

	s.InitTestNode()
	s.StartTestNode()
	defer func() {
		s.StopTestNode()
	}()

	s.waitForConsensus(1)

	activeNodes := s.fixture().bootstrapNodes[0].serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(s.getNodesCount(), len(activeNodes))

	s.waitForConsensus(1)

	activeNodes = s.fixture().bootstrapNodes[0].serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(s.getNodesCount()+1, len(activeNodes))

	s.fixture().testNode.serviceNetwork.GracefulStop(context.Background())

	s.waitForConsensus(2)

	activeNodes = s.fixture().bootstrapNodes[0].serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(s.getNodesCount(), len(activeNodes))
}

func TestServiceNetworkOneBootstrap(t *testing.T) {
	s := NewTestSuite(1, 0)
	suite.Run(t, s)
}

func TestServiceNetworkManyBootstraps(t *testing.T) {
	s := NewTestSuite(15, 0)
	suite.Run(t, s)
}

func TestServiceNetworkManyNodes(t *testing.T) {
	t.Skip("tmp 123")

	s := NewTestSuite(5, 10)
	suite.Run(t, s)
}

// Full timeout test
type FullTimeoutPhaseManager struct {
}

func (ftpm *FullTimeoutPhaseManager) OnPulse(ctx context.Context, pulse *core.Pulse, pulseStartTime time.Time) error {
	return nil
}

func (s *testSuite) TestFullTimeOut() {
	if len(s.fixture().bootstrapNodes) < consensusMin {
		s.T().Skip(consensusMinMsg)
	}

	// TODO: make this set operation thread-safe somehow (race detector does not like this code)
	wrapper := s.fixture().bootstrapNodes[1].serviceNetwork.PhaseManager.(*phaseManagerWrapper)
	wrapper.original = &FullTimeoutPhaseManager{}
	s.fixture().bootstrapNodes[1].serviceNetwork.PhaseManager = wrapper

	s.waitForConsensus(2)

	activeNodes := s.fixture().bootstrapNodes[0].serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(s.getNodesCount()-1, len(activeNodes))
}

// Partial timeout

func (s *testSuite) TestPartialPositive1PhaseTimeOut() {
	if len(s.fixture().bootstrapNodes) < consensusMin {
		s.T().Skip(consensusMinMsg)
	}

	setCommunicatorMock(s.fixture().bootstrapNodes, PartialPositive1Phase)

	s.waitForConsensusExcept(2, s.fixture().bootstrapNodes[0].id)
	activeNodes := s.fixture().bootstrapNodes[1].serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(s.getNodesCount(), len(activeNodes))
}

func (s *testSuite) TestPartialPositive2PhaseTimeOut() {
	if len(s.fixture().bootstrapNodes) < consensusMin {
		s.T().Skip(consensusMinMsg)
	}

	setCommunicatorMock(s.fixture().bootstrapNodes, PartialPositive2Phase)

	s.waitForConsensusExcept(2, s.fixture().bootstrapNodes[0].id)
	activeNodes := s.fixture().bootstrapNodes[1].serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(s.getNodesCount(), len(activeNodes))
}

func (s *testSuite) TestPartialNegative1PhaseTimeOut() {
	if len(s.fixture().bootstrapNodes) < consensusMin {
		s.T().Skip(consensusMinMsg)
	}

	setCommunicatorMock(s.fixture().bootstrapNodes, PartialNegative1Phase)

	s.waitForConsensusExcept(2, s.fixture().bootstrapNodes[0].id)
	activeNodes := s.fixture().bootstrapNodes[1].serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(s.getNodesCount()-1, len(activeNodes))
}

func (s *testSuite) TestPartialNegative2PhaseTimeOut() {
	if len(s.fixture().bootstrapNodes) < consensusMin {
		s.T().Skip(consensusMinMsg)
	}

	setCommunicatorMock(s.fixture().bootstrapNodes, PartialNegative2Phase)

	s.waitForConsensusExcept(2, s.fixture().bootstrapNodes[0].id)
	activeNodes := s.fixture().bootstrapNodes[1].serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(s.getNodesCount(), len(activeNodes))
}

func (s *testSuite) TestPartialNegative3PhaseTimeOut() {
	if len(s.fixture().bootstrapNodes) < consensusMin {
		s.T().Skip(consensusMinMsg)
	}

	setCommunicatorMock(s.fixture().bootstrapNodes, PartialNegative3Phase)

	s.waitForConsensusExcept(2, s.fixture().bootstrapNodes[0].id)
	activeNodes := s.fixture().bootstrapNodes[1].serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(s.getNodesCount(), len(activeNodes))
}

func (s *testSuite) TestPartialPositive3PhaseTimeOut() {
	if len(s.fixture().bootstrapNodes) < consensusMin {
		s.T().Skip(consensusMinMsg)
	}

	setCommunicatorMock(s.fixture().bootstrapNodes, PartialPositive3Phase)

	s.waitForConsensusExcept(2, s.fixture().bootstrapNodes[0].id)
	activeNodes := s.fixture().bootstrapNodes[1].serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(s.getNodesCount(), len(activeNodes))
}

func (s *testSuite) TestPartialNegative23PhaseTimeOut() {
	if len(s.fixture().bootstrapNodes) < consensusMin {
		s.T().Skip(consensusMinMsg)
	}

	setCommunicatorMock(s.fixture().bootstrapNodes, PartialNegative23Phase)

	s.waitForConsensusExcept(2, s.fixture().bootstrapNodes[0].id)
	activeNodes := s.fixture().bootstrapNodes[1].serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(s.getNodesCount(), len(activeNodes))
}

func (s *testSuite) TestPartialPositive23PhaseTimeOut() {
	if len(s.fixture().bootstrapNodes) < consensusMin {
		s.T().Skip(consensusMinMsg)
	}

	setCommunicatorMock(s.fixture().bootstrapNodes, PartialPositive23Phase)

	s.waitForConsensusExcept(2, s.fixture().bootstrapNodes[0].id)
	activeNodes := s.fixture().bootstrapNodes[1].serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(s.getNodesCount(), len(activeNodes))
}

func (s *testSuite) TestDiscoveryDown() {
	if len(s.fixture().bootstrapNodes) < consensusMin {
		s.T().Skip(consensusMinMsg)
	}

	s.fixture().bootstrapNodes[0].serviceNetwork.Stop(context.Background())
	s.waitForConsensusExcept(2, s.fixture().bootstrapNodes[0].id)
	activeNodes := s.fixture().bootstrapNodes[1].serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(s.getNodesCount()-1, len(activeNodes))
}

func (s *testSuite) TestDiscoveryRestart() {
	if len(s.fixture().bootstrapNodes) < consensusMin {
		s.T().Skip(consensusMinMsg)
	}

	s.waitForConsensus(1)

	log.Info("Discovery node stopping...")
	err := s.fixture().bootstrapNodes[0].serviceNetwork.Stop(context.Background())
	s.fixture().bootstrapNodes[0].serviceNetwork.NodeKeeper.(*nodeKeeperWrapper).Wipe(true)
	log.Info("Discovery node stopped...")
	require.NoError(s.T(), err)

	s.waitForConsensusExcept(2, s.fixture().bootstrapNodes[0].id)
	activeNodes := s.fixture().bootstrapNodes[1].serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(s.getNodesCount()-1, len(activeNodes))

	log.Info("Discovery node starting...")
	err = s.fixture().bootstrapNodes[0].serviceNetwork.Start(context.Background())
	log.Info("Discovery node started")
	require.NoError(s.T(), err)

	s.waitForConsensusExcept(3, s.fixture().bootstrapNodes[0].id)
	activeNodes = s.fixture().bootstrapNodes[1].serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(s.getNodesCount(), len(activeNodes))
	activeNodes = s.fixture().bootstrapNodes[0].serviceNetwork.NodeKeeper.GetActiveNodes()
	s.Equal(s.getNodesCount(), len(activeNodes))
}

func setCommunicatorMock(nodes []*networkNode, opt CommunicatorTestOpt) {
	ref := nodes[0].id
	timedOutNodesCount := 0
	switch opt {
	case PartialNegative1Phase, PartialNegative2Phase, PartialNegative3Phase, PartialNegative23Phase:
		timedOutNodesCount = int(float64(len(nodes)) * 0.6)
	case PartialPositive1Phase, PartialPositive2Phase, PartialPositive3Phase, PartialPositive23Phase:
		timedOutNodesCount = int(float64(len(nodes)) * 0.2)
	}
	// TODO: make these set operations thread-safe somehow (race detector does not like this code)
	for i := 1; i <= timedOutNodesCount; i++ {
		comm := nodes[i].serviceNetwork.PhaseManager.(*phaseManagerWrapper).original.(*phases.Phases).FirstPhase.(*phases.FirstPhaseImpl).Communicator
		wrapper := &CommunicatorMock{communicator: comm, ignoreFrom: ref, testOpt: opt}
		phasemanager := nodes[i].serviceNetwork.PhaseManager.(*phaseManagerWrapper).original.(*phases.Phases)
		phasemanager.FirstPhase.(*phases.FirstPhaseImpl).Communicator = wrapper
		phasemanager.SecondPhase.(*phases.SecondPhaseImpl).Communicator = wrapper
		phasemanager.ThirdPhase.(*phases.ThirdPhaseImpl).Communicator = wrapper
	}
}
