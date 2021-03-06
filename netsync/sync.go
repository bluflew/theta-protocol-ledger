package netsync

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/util"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/dispatcher"
	"github.com/thetatoken/theta/p2p"
	p2ptypes "github.com/thetatoken/theta/p2p/types"
	"github.com/thetatoken/theta/rlp"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "netsync"})

type MessageConsumer interface {
	AddMessage(interface{})
}

var _ p2p.MessageHandler = (*SyncManager)(nil)

// SyncManager is an intermediate layer between consensus engine and p2p network. Its main responsibilities are to manage
// fast blocks sync among peers and buffer orphaned block/CC. Otherwise messages are passed through to consensus engine.
type SyncManager struct {
	chain      *blockchain.Chain
	consensus  core.ConsensusEngine
	consumer   MessageConsumer
	dispatcher *dispatcher.Dispatcher
	requestMgr *RequestManager

	wg      *sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool

	incoming chan p2ptypes.Message

	logger *log.Entry
}

func NewSyncManager(chain *blockchain.Chain, cons core.ConsensusEngine, network p2p.Network, disp *dispatcher.Dispatcher, consumer MessageConsumer) *SyncManager {
	sm := &SyncManager{
		chain:      chain,
		consensus:  cons,
		consumer:   consumer,
		dispatcher: disp,

		wg:       &sync.WaitGroup{},
		incoming: make(chan p2ptypes.Message, viper.GetInt(common.CfgSyncMessageQueueSize)),
	}
	sm.requestMgr = NewRequestManager(sm)
	network.RegisterMessageHandler(sm)

	logger := util.GetLoggerForModule("sync")
	if viper.GetBool(common.CfgLogPrintSelfID) {
		logger = logger.WithFields(log.Fields{"id": sm.consensus.ID()})
	}
	sm.logger = logger

	return sm
}

func (sm *SyncManager) Start(ctx context.Context) {
	c, cancel := context.WithCancel(ctx)
	sm.ctx = c
	sm.cancel = cancel

	sm.requestMgr.Start(c)

	sm.wg.Add(1)
	go sm.mainLoop()
}

func (sm *SyncManager) Stop() {
	sm.cancel()
}

func (sm *SyncManager) Wait() {
	sm.requestMgr.Wait()
	sm.wg.Wait()
}

func (sm *SyncManager) mainLoop() {
	defer sm.wg.Done()

	for {
		select {
		case <-sm.ctx.Done():
			sm.stopped = true
			return
		case msg := <-sm.incoming:
			sm.processMessage(msg)
		}
	}
}

// GetChannelIDs implements the p2p.MessageHandler interface.
func (sm *SyncManager) GetChannelIDs() []common.ChannelIDEnum {
	return []common.ChannelIDEnum{
		common.ChannelIDHeader,
		common.ChannelIDBlock,
		common.ChannelIDProposal,
		common.ChannelIDCC,
		common.ChannelIDVote,
	}
}

// ParseMessage implements p2p.MessageHandler interface.
func (sm *SyncManager) ParseMessage(peerID string, channelID common.ChannelIDEnum,
	rawMessageBytes common.Bytes) (p2ptypes.Message, error) {
	message := p2ptypes.Message{
		PeerID:    peerID,
		ChannelID: channelID,
	}
	data, err := decodeMessage(rawMessageBytes)
	message.Content = data
	return message, err
}

// EncodeMessage implements p2p.MessageHandler interface.
func (sm *SyncManager) EncodeMessage(message interface{}) (common.Bytes, error) {
	return encodeMessage(message)
}

// HandleMessage implements p2p.MessageHandler interface.
func (sm *SyncManager) HandleMessage(msg p2ptypes.Message) (err error) {
	sm.incoming <- msg
	return
}

func (sm *SyncManager) processMessage(message p2ptypes.Message) {
	switch content := message.Content.(type) {
	case dispatcher.InventoryRequest:
		sm.handleInvRequest(message.PeerID, &content)
	case dispatcher.InventoryResponse:
		sm.handleInvResponse(message.PeerID, &content)
	case dispatcher.DataRequest:
		sm.handleDataRequest(message.PeerID, &content)
	case dispatcher.DataResponse:
		sm.handleDataResponse(message.PeerID, &content)
	default:
		sm.logger.WithFields(log.Fields{
			"message": message,
		}).Warn("Received unknown message")
	}
}

// PassdownMessage passes message through to the consumer.
func (sm *SyncManager) PassdownMessage(msg interface{}) {
	sm.consumer.AddMessage(msg)
}

// locateStart finds first start hash that exists in local chain.
func (m *SyncManager) locateStart(starts []string) common.Hash {
	var start common.Hash
	for i := 0; i < len(starts); i++ {
		curr := common.HexToHash(starts[i])
		if _, err := m.chain.FindBlock(curr); err == nil {
			start = curr
			break
		}
	}
	return start
}

// Dump blocks from start until end or MaxInventorySize is reached.
func (m *SyncManager) collectBlocks(start common.Hash, end common.Hash) []string {
	ret := []string{}

	lfbHeight := m.consensus.GetLastFinalizedBlock().Height
	q := []common.Hash{start}
	for len(q) > 0 && len(ret) < dispatcher.MaxInventorySize-1 {
		curr := q[0]
		q = q[1:]
		block, err := m.chain.FindBlock(curr)
		if err != nil {
			m.logger.WithFields(log.Fields{
				"hash": curr.Hex(),
			}).Debug("Failed to find block with given hash")
			return ret
		}
		ret = append(ret, curr.Hex())
		if curr == end {
			break
		}

		if block.Height < lfbHeight {
			// Enqueue finalized child.
			for _, child := range block.Children {
				block, err := m.chain.FindBlock(child)
				if err != nil {
					m.logger.WithFields(log.Fields{
						"err":  err,
						"hash": curr,
					}).Debug("Failed to load block")
					return ret
				}
				if block.Status.IsFinalized() {
					q = append(q, block.Hash())
					break
				}
			}
		} else {
			// Enqueue all children.
			q = append(q, block.Children...)
		}
	}

	// Make sure response is in size limit.
	if len(ret) > dispatcher.MaxInventorySize {
		ret = ret[:dispatcher.MaxInventorySize-1]
	}

	// Add last finalized block in the end so that receiver is aware of latest network state.
	ret = append(ret, m.consensus.GetLastFinalizedBlock().Hash().Hex())

	return ret
}

func (m *SyncManager) handleInvRequest(peerID string, req *dispatcher.InventoryRequest) {
	m.logger.WithFields(log.Fields{
		"channelID":   req.ChannelID,
		"startHashes": req.Starts,
		"endHash":     req.End,
		"peerID":      peerID,
	}).Debug("Received inventory request")

	switch req.ChannelID {
	case common.ChannelIDBlock:

		start := m.locateStart(req.Starts)
		if start.IsEmpty() {
			m.logger.WithFields(log.Fields{
				"channelID": req.ChannelID,
				"peerID":    peerID,
			}).Debug("No start hash can be found in local chain")
			return
		}

		end := common.HexToHash(req.End)
		blocks := m.collectBlocks(start, end)

		// Send response.
		resp := dispatcher.InventoryResponse{ChannelID: common.ChannelIDBlock, Entries: blocks}
		m.logger.WithFields(log.Fields{
			"channelID":         resp.ChannelID,
			"len(resp.Entries)": len(resp.Entries),
			"peerID":            peerID,
		}).Debug("Sending inventory response")
		m.dispatcher.SendInventory([]string{peerID}, resp)
	default:
		m.logger.WithFields(log.Fields{"channelID": req.ChannelID}).Warn("Unsupported channelID in received InvRequest")
	}

}

func (m *SyncManager) handleInvResponse(peerID string, resp *dispatcher.InventoryResponse) {
	m.logger.WithFields(log.Fields{
		"channelID":   resp.ChannelID,
		"InvResponse": resp,
		"peerID":      peerID,
	}).Debug("Received Inventory Response")

	switch resp.ChannelID {
	case common.ChannelIDBlock:
		for _, hashStr := range resp.Entries {
			hash := common.HexToHash(hashStr)
			m.requestMgr.AddHash(hash, []string{peerID})
		}
	default:
		m.logger.WithFields(log.Fields{
			"channelID": resp.ChannelID,
			"peerID":    peerID,
		}).Warn("Unsupported channelID in received Inventory Request")
	}
}

func (m *SyncManager) handleDataRequest(peerID string, data *dispatcher.DataRequest) {
	switch data.ChannelID {
	case common.ChannelIDBlock:
		for _, hashStr := range data.Entries {
			hash := common.HexToHash(hashStr)
			block, err := m.chain.FindBlock(hash)
			if err != nil {
				m.logger.WithFields(log.Fields{
					"channelID": data.ChannelID,
					"hashStr":   hashStr,
					"err":       err,
					"peerID":    peerID,
				}).Debug("Failed to find hash string locally")
				return
			}

			payload, err := rlp.EncodeToBytes(block.Block)
			if err != nil {
				m.logger.WithFields(log.Fields{
					"block":  block,
					"peerID": peerID,
				}).Error("Failed to encode block")
				return
			}
			data := dispatcher.DataResponse{
				ChannelID: common.ChannelIDBlock,
				Payload:   payload,
			}
			m.logger.WithFields(log.Fields{
				"channelID": data.ChannelID,
				"hashStr":   hashStr,
				"peerID":    peerID,
			}).Debug("Sending requested block")
			m.dispatcher.SendData([]string{peerID}, data)
		}
	default:
		m.logger.WithFields(log.Fields{
			"channelID": data.ChannelID,
		}).Warn("Unsupported channelID in received DataRequest")
	}
}

func (m *SyncManager) handleDataResponse(peerID string, data *dispatcher.DataResponse) {
	switch data.ChannelID {
	case common.ChannelIDBlock:
		block := core.NewBlock()
		err := rlp.DecodeBytes(data.Payload, block)
		if err != nil {
			m.logger.WithFields(log.Fields{
				"channelID": data.ChannelID,
				"payload":   data.Payload,
				"error":     err,
				"peerID":    peerID,
			}).Warn("Failed to decode DataResponse payload")
			return
		}
		m.handleBlock(block)
	case common.ChannelIDVote:
		vote := core.Vote{}
		err := rlp.DecodeBytes(data.Payload, &vote)
		if err != nil {
			m.logger.WithFields(log.Fields{
				"channelID": data.ChannelID,
				"payload":   data.Payload,
				"error":     err,
				"peerID":    peerID,
			}).Warn("Failed to decode DataResponse payload")
			return
		}
		m.handleVote(vote)
	case common.ChannelIDProposal:
		proposal := &core.Proposal{}
		err := rlp.DecodeBytes(data.Payload, proposal)
		if err != nil {
			m.logger.WithFields(log.Fields{
				"channelID": data.ChannelID,
				"payload":   data.Payload,
				"error":     err,
				"peerID":    peerID,
			}).Warn("Failed to decode DataResponse payload")
			return
		}
		m.handleProposal(proposal)
	default:
		m.logger.WithFields(log.Fields{
			"channelID": data.ChannelID,
		}).Warn("Unsupported channelID in received DataResponse")
	}
}

func (sm *SyncManager) handleProposal(p *core.Proposal) {
	sm.logger.WithFields(log.Fields{
		"proposal": p,
	}).Debug("Received proposal")

	if p.Votes != nil {
		for _, vote := range p.Votes.Votes() {
			sm.handleVote(vote)
		}
	}
	sm.handleBlock(p.Block)
}

func (sm *SyncManager) handleBlock(block *core.Block) {
	sm.logger.WithFields(log.Fields{
		"block.Hash":   block.Hash().Hex(),
		"block.Parent": block.Parent.Hex(),
	}).Debug("Received block")

	if eb, err := sm.chain.FindBlock(block.Hash()); err == nil && !eb.Status.IsPending() {
		return
	}

	sm.requestMgr.AddBlock(block)

	sm.dispatcher.SendInventory([]string{}, dispatcher.InventoryResponse{
		ChannelID: common.ChannelIDBlock,
		Entries:   []string{block.Hash().Hex()},
	})
}

func (sm *SyncManager) handleVote(vote core.Vote) {
	sm.logger.WithFields(log.Fields{
		"vote.Hash":  vote.Block.Hex(),
		"vote.ID":    vote.ID.Hex(),
		"vote.Epoch": vote.Epoch,
	}).Debug("Received vote")

	votes := sm.chain.FindVotesByHash(vote.Block).Votes()
	for _, v := range votes {
		// Check if vote already processed.
		if v.Block == vote.Block && v.Epoch == vote.Epoch && v.Height == vote.Height && v.ID == vote.ID {
			return
		}
	}

	sm.PassdownMessage(vote)

	payload, err := rlp.EncodeToBytes(vote)
	if err != nil {
		sm.logger.WithFields(log.Fields{"vote": vote}).Error("Failed to encode vote")
		return
	}
	msg := dispatcher.DataResponse{
		ChannelID: common.ChannelIDVote,
		Payload:   payload,
	}
	sm.dispatcher.SendData([]string{}, msg)
}
