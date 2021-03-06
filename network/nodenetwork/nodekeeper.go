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

package nodenetwork

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/insolar/insolar/instrumentation/inslogger"

	"github.com/insolar/insolar/configuration"
	"github.com/insolar/insolar/consensus"
	consensusPackets "github.com/insolar/insolar/consensus/packets"
	"github.com/insolar/insolar/core"
	"github.com/insolar/insolar/log"
	"github.com/insolar/insolar/network"
	"github.com/insolar/insolar/network/transport"
	"github.com/insolar/insolar/network/transport/host"
	"github.com/insolar/insolar/network/utils"
	"github.com/insolar/insolar/version"
	"github.com/pkg/errors"
	"go.opencensus.io/stats"
)

// NewNodeNetwork create active node component
func NewNodeNetwork(configuration configuration.HostNetwork, certificate core.Certificate) (core.NodeNetwork, error) {
	origin, err := createOrigin(configuration, certificate)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create origin node")
	}
	nodeKeeper := NewNodeKeeper(origin)
	nodeKeeper.SetState(network.Waiting)
	if len(certificate.GetDiscoveryNodes()) == 0 || utils.OriginIsDiscovery(certificate) {
		nodeKeeper.SetState(network.Ready)
		nodeKeeper.AddActiveNodes([]core.Node{origin})
	}
	return nodeKeeper, nil
}

func createOrigin(configuration configuration.HostNetwork, certificate core.Certificate) (MutableNode, error) {
	publicAddress, err := resolveAddress(configuration)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to resolve public address")
	}

	role := certificate.GetRole()
	if role == core.StaticRoleUnknown {
		log.Info("[ createOrigin ] Use core.StaticRoleLightMaterial, since no role in certificate")
		role = core.StaticRoleLightMaterial
	}

	return newMutableNode(
		*certificate.GetNodeRef(),
		role,
		certificate.GetPublicKey(),
		publicAddress,
		version.Version,
	), nil
}

func resolveAddress(configuration configuration.HostNetwork) (string, error) {
	conn, address, err := transport.NewConnection(configuration.Transport)
	if err != nil {
		return "", err
	}
	err = conn.Close()
	if err != nil {
		log.Warn(err)
	}
	return address, nil
}

// NewNodeKeeper create new NodeKeeper
func NewNodeKeeper(origin core.Node) network.NodeKeeper {
	result := &nodekeeper{
		origin:       origin,
		state:        network.Undefined,
		claimQueue:   newClaimQueue(),
		active:       make(map[core.RecordRef]core.Node),
		indexNode:    make(map[core.StaticRole]*recordRefSet),
		indexShortID: make(map[core.ShortNodeID]core.Node),
		tempMapR:     make(map[core.RecordRef]*host.Host),
		tempMapS:     make(map[core.ShortNodeID]*host.Host),
		sync:         newUnsyncList(origin, []core.Node{}, 0),
	}
	result.SetState(network.Ready)
	return result
}

type nodekeeper struct {
	origin     core.Node
	claimQueue *claimQueue

	nodesJoinedDuringPrevPulse bool

	cloudHashLock sync.RWMutex
	cloudHash     []byte

	activeLock   sync.RWMutex
	active       map[core.RecordRef]core.Node
	indexNode    map[core.StaticRole]*recordRefSet
	indexShortID map[core.ShortNodeID]core.Node

	tempLock sync.RWMutex
	tempMapR map[core.RecordRef]*host.Host
	tempMapS map[core.ShortNodeID]*host.Host

	syncLock sync.Mutex
	sync     network.UnsyncList
	state    network.NodeKeeperState

	isBootstrap     bool
	isBootstrapLock sync.RWMutex

	Cryptography core.CryptographyService `inject:""`
	Handler      core.TerminationHandler  `inject:""`
}

func (nk *nodekeeper) Wipe(isDiscovery bool) {
	log.Warn("don't use it in production")

	nk.isBootstrapLock.Lock()
	nk.isBootstrap = false
	nk.isBootstrapLock.Unlock()

	nk.tempLock.Lock()
	nk.tempMapR = make(map[core.RecordRef]*host.Host)
	nk.tempMapS = make(map[core.ShortNodeID]*host.Host)
	nk.tempLock.Unlock()

	nk.cloudHashLock.Lock()
	nk.cloudHash = nil
	nk.cloudHashLock.Unlock()

	nk.activeLock.Lock()
	defer nk.activeLock.Unlock()

	nk.claimQueue = newClaimQueue()
	nk.nodesJoinedDuringPrevPulse = false
	nk.active = make(map[core.RecordRef]core.Node)
	nk.reindex()
	nk.syncLock.Lock()
	nk.sync = newUnsyncList(nk.origin, []core.Node{}, 0)
	if isDiscovery {
		nk.addActiveNode(nk.origin)
		nk.state = network.Ready
	}
	nk.syncLock.Unlock()
}

func (nk *nodekeeper) AddTemporaryMapping(nodeID core.RecordRef, shortID core.ShortNodeID, address string) error {
	consensusAddress, err := incrementPort(address)
	if err != nil {
		return errors.Wrapf(err, "Failed to increment port for address %s", address)
	}
	h, err := host.NewHostNS(consensusAddress, nodeID, shortID)
	if err != nil {
		return errors.Wrapf(err, "Failed to generate address (%s, %s, %d)", consensusAddress, nodeID, shortID)
	}
	nk.tempLock.Lock()
	nk.tempMapR[nodeID] = h
	nk.tempMapS[shortID] = h
	nk.tempLock.Unlock()
	log.Infof("Added temporary mapping: %s -> (%s, %d)", consensusAddress, nodeID, shortID)
	return nil
}

func (nk *nodekeeper) ResolveConsensus(shortID core.ShortNodeID) *host.Host {
	nk.tempLock.RLock()
	defer nk.tempLock.RUnlock()

	return nk.tempMapS[shortID]
}

func (nk *nodekeeper) ResolveConsensusRef(nodeID core.RecordRef) *host.Host {
	nk.tempLock.RLock()
	defer nk.tempLock.RUnlock()

	return nk.tempMapR[nodeID]
}

// TODO: remove this method when bootstrap mechanism completed
// IsBootstrapped method returns true when bootstrapNodes are connected to each other
func (nk *nodekeeper) IsBootstrapped() bool {
	nk.isBootstrapLock.RLock()
	defer nk.isBootstrapLock.RUnlock()

	return nk.isBootstrap
}

// TODO: remove this method when bootstrap mechanism completed
// SetIsBootstrapped method set is bootstrap completed
func (nk *nodekeeper) SetIsBootstrapped(isBootstrap bool) {
	nk.isBootstrapLock.Lock()
	defer nk.isBootstrapLock.Unlock()

	nk.isBootstrap = isBootstrap
}

func (nk *nodekeeper) GetOrigin() core.Node {
	nk.activeLock.RLock()
	defer nk.activeLock.RUnlock()

	return nk.origin
}

func (nk *nodekeeper) GetCloudHash() []byte {
	nk.cloudHashLock.RLock()
	defer nk.cloudHashLock.RUnlock()

	return nk.cloudHash
}

func (nk *nodekeeper) SetCloudHash(cloudHash []byte) {
	nk.cloudHashLock.Lock()
	defer nk.cloudHashLock.Unlock()

	nk.cloudHash = cloudHash
}

func (nk *nodekeeper) GetActiveNodes() []core.Node {
	nk.activeLock.RLock()
	result := make([]core.Node, len(nk.active))
	index := 0
	for _, node := range nk.active {
		result[index] = node
		index++
	}
	nk.activeLock.RUnlock()
	// Sort active nodes to return list with determinate order on every node.
	// If we have more than 10k nodes, we need to optimize this
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID().Compare(result[j].ID()) < 0
	})
	return result
}

func (nk *nodekeeper) GetActiveNodesByRole(role core.DynamicRole) []core.RecordRef {
	nk.activeLock.RLock()
	defer nk.activeLock.RUnlock()

	list, exists := nk.indexNode[jetRoleToNodeRole(role)]
	if !exists {
		return nil
	}
	return list.Collect()
}

func (nk *nodekeeper) AddActiveNodes(nodes []core.Node) {
	nk.activeLock.Lock()
	defer nk.activeLock.Unlock()

	activeNodes := make([]string, len(nodes))
	for i, node := range nodes {
		nk.addActiveNode(node)
		activeNodes[i] = node.ID().String()
	}
	syncList := nk.sync.(*unsyncList)
	syncList.addNodes(nodes)
	log.Debugf("Added active nodes: %s", strings.Join(activeNodes, ", "))
}

func (nk *nodekeeper) GetActiveNode(ref core.RecordRef) core.Node {
	nk.activeLock.RLock()
	defer nk.activeLock.RUnlock()

	return nk.active[ref]
}

func (nk *nodekeeper) GetActiveNodeByShortID(shortID core.ShortNodeID) core.Node {
	nk.activeLock.RLock()
	defer nk.activeLock.RUnlock()

	return nk.indexShortID[shortID]
}

func (nk *nodekeeper) addActiveNode(node core.Node) {
	if node.ID().Equal(nk.origin.ID()) {
		nk.origin = node
		log.Infof("Added origin node %s to active list", nk.origin.ID())
	}
	nk.active[node.ID()] = node

	nk.addToIndex(node)
}

func (nk *nodekeeper) addToIndex(node core.Node) {
	list, ok := nk.indexNode[node.Role()]
	if !ok {
		list = newRecordRefSet()
	}
	list.Add(node.ID())
	nk.indexNode[node.Role()] = list

	nk.indexShortID[node.ShortID()] = node
}

func (nk *nodekeeper) SetState(state network.NodeKeeperState) {
	nk.syncLock.Lock()
	defer nk.syncLock.Unlock()

	nk.state = state
}

func (nk *nodekeeper) GetState() network.NodeKeeperState {
	nk.syncLock.Lock()
	defer nk.syncLock.Unlock()

	return nk.state
}

func (nk *nodekeeper) GetOriginJoinClaim() (*consensusPackets.NodeJoinClaim, error) {
	nk.activeLock.RLock()
	defer nk.activeLock.RUnlock()

	return nk.nodeToSignedClaim()
}

func (nk *nodekeeper) GetOriginAnnounceClaim(mapper consensusPackets.BitSetMapper) (*consensusPackets.NodeAnnounceClaim, error) {
	nk.activeLock.RLock()
	defer nk.activeLock.RUnlock()

	return nk.nodeToAnnounceClaim(mapper)
}

func (nk *nodekeeper) AddPendingClaim(claim consensusPackets.ReferendumClaim) bool {
	nk.claimQueue.Push(claim)
	return true
}

func (nk *nodekeeper) GetClaimQueue() network.ClaimQueue {
	return nk.claimQueue
}

func (nk *nodekeeper) NodesJoinedDuringPreviousPulse() bool {
	nk.activeLock.RLock()
	defer nk.activeLock.RUnlock()

	return nk.nodesJoinedDuringPrevPulse
}

func (nk *nodekeeper) GetUnsyncList() network.UnsyncList {
	activeNodes := nk.GetActiveNodes()
	return newUnsyncList(nk.origin, activeNodes, len(activeNodes))
}

func (nk *nodekeeper) GetSparseUnsyncList(length int) network.UnsyncList {
	return newSparseUnsyncList(nk.origin, length)
}

func (nk *nodekeeper) Sync(list network.UnsyncList) {
	nk.syncLock.Lock()
	defer nk.syncLock.Unlock()

	nodes := list.GetActiveNodes()

	foundOrigin := false
	for _, node := range nodes {
		if node.ID().Equal(nk.origin.ID()) {
			foundOrigin = true
			nk.state = network.Ready
		}
	}

	if nk.shouldExit(foundOrigin) {
		nk.gracefullyStop()
	}

	nk.sync = list
}

func (nk *nodekeeper) MoveSyncToActive(ctx context.Context) error {
	nk.activeLock.Lock()
	nk.syncLock.Lock()
	defer func() {
		nk.syncLock.Unlock()
		nk.activeLock.Unlock()
	}()

	nk.tempLock.Lock()
	// clear temporary mappings
	nk.tempMapR = make(map[core.RecordRef]*host.Host)
	nk.tempMapS = make(map[core.ShortNodeID]*host.Host)
	nk.tempLock.Unlock()

	mergeResult, err := nk.sync.GetMergedCopy()
	if err != nil {
		return errors.Wrap(err, "[ MoveSyncToActive ] Failed to calculate new active list")
	}
	if mergeResult.Flags.ShouldExit {
		nk.gracefullyStop()
	}

	inslogger.FromContext(ctx).Infof("[ MoveSyncToActive ] New active list confirmed. Active list size: %d -> %d",
		len(nk.active), len(mergeResult.ActiveList))
	nk.active = mergeResult.ActiveList
	stats.Record(ctx, consensus.ActiveNodes.M(int64(len(nk.active))))
	nk.reindex()
	nk.nodesJoinedDuringPrevPulse = mergeResult.Flags.NodesJoinedDuringPrevPulse
	return nil
}

func (nk *nodekeeper) gracefullyStop() {
	// TODO: graceful stop
	nk.Handler.Abort()
}

func (nk *nodekeeper) reindex() {
	// drop all indexes
	nk.indexNode = make(map[core.StaticRole]*recordRefSet)
	nk.indexShortID = make(map[core.ShortNodeID]core.Node)

	for _, node := range nk.active {
		nk.addToIndex(node)
	}
}

func (nk *nodekeeper) shouldExit(foundOrigin bool) bool {
	return !foundOrigin && nk.state == network.Ready && len(nk.active) != 0
}

func (nk *nodekeeper) nodeToSignedClaim() (*consensusPackets.NodeJoinClaim, error) {
	claim, err := consensusPackets.NodeToClaim(nk.origin)
	if err != nil {
		return nil, err
	}
	dataToSign, err := claim.SerializeRaw()
	log.Debugf("dataToSign len: %d", len(dataToSign))
	if err != nil {
		return nil, errors.Wrap(err, "[ nodeToSignedClaim ] failed to serialize a claim")
	}
	sign, err := nk.sign(dataToSign)
	log.Debugf("sign len: %d", len(sign))
	if err != nil {
		return nil, errors.Wrap(err, "[ nodeToSignedClaim ] failed to sign a claim")
	}
	copy(claim.Signature[:], sign[:consensusPackets.SignatureLength])
	return claim, nil
}

func (nk *nodekeeper) nodeToAnnounceClaim(mapper consensusPackets.BitSetMapper) (*consensusPackets.NodeAnnounceClaim, error) {
	claim := consensusPackets.NodeAnnounceClaim{}
	joinClaim, err := consensusPackets.NodeToClaim(nk.origin)
	if err != nil {
		return nil, err
	}
	claim.NodeJoinClaim = *joinClaim
	claim.NodeCount = uint16(mapper.Length())
	announcerIndex, err := mapper.RefToIndex(nk.origin.ID())
	if err != nil {
		return nil, errors.Wrap(err, "[ nodeToAnnounceClaim ] failed to map origin node ID to bitset index")
	}
	claim.NodeAnnouncerIndex = uint16(announcerIndex)
	claim.BitSetMapper = mapper
	claim.SetCloudHash(nk.GetCloudHash())
	return &claim, nil
}

func (nk *nodekeeper) sign(data []byte) ([]byte, error) {
	sign, err := nk.Cryptography.Sign(data)
	if err != nil {
		return nil, errors.Wrap(err, "[ sign ] failed to sign a claim")
	}
	return sign.Bytes(), nil
}

func jetRoleToNodeRole(role core.DynamicRole) core.StaticRole {
	switch role {
	case core.DynamicRoleVirtualExecutor:
		return core.StaticRoleVirtual
	case core.DynamicRoleVirtualValidator:
		return core.StaticRoleVirtual
	case core.DynamicRoleLightExecutor:
		return core.StaticRoleLightMaterial
	case core.DynamicRoleLightValidator:
		return core.StaticRoleLightMaterial
	case core.DynamicRoleHeavyExecutor:
		return core.StaticRoleHeavyMaterial
	default:
		return core.StaticRoleUnknown
	}
}
