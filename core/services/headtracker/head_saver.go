package headtracker

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/chainlink/core/logger"
	"github.com/smartcontractkit/chainlink/core/services/eth"
	httypes "github.com/smartcontractkit/chainlink/core/services/headtracker/types"
)

type headSaver struct {
	orm    ORM
	config Config
	logger logger.Logger
	heads  Heads
}

func NewHeadSaver(lggr logger.Logger, orm ORM, config Config) httypes.HeadSaver {
	return &headSaver{
		orm:    orm,
		config: config,
		logger: lggr.Named(logger.HeadSaver),
		heads:  NewHeads(),
	}
}

func (hs *headSaver) Save(ctx context.Context, head *eth.Head) error {
	if err := hs.orm.IdempotentInsertHead(ctx, head); err != nil {
		return err
	}

	historyDepth := uint(hs.config.EvmHeadTrackerHistoryDepth())
	hs.heads.AddHeads(historyDepth, head)

	return hs.orm.TrimOldHeads(ctx, historyDepth)
}

func (hs *headSaver) LoadFromDB(ctx context.Context) (chain *eth.Head, err error) {
	historyDepth := uint(hs.config.EvmHeadTrackerHistoryDepth())
	heads, err := hs.orm.LatestHeads(ctx, historyDepth)
	if err != nil {
		return nil, err
	}

	hs.heads.AddHeads(historyDepth, heads...)
	return hs.heads.LatestHead(), nil
}

func (hs *headSaver) LatestHeadFromDB(ctx context.Context) (head *eth.Head, err error) {
	return hs.orm.LatestHead(ctx)
}

func (hs *headSaver) LatestChain() *eth.Head {
	head := hs.heads.LatestHead()
	if head == nil {
		return nil
	}
	if head.ChainLength() < hs.config.EvmFinalityDepth() {
		hs.logger.Debugw("chain shorter than EvmFinalityDepth", "chainLen", head.ChainLength(), "evmFinalityDepth", hs.config.EvmFinalityDepth())
	}
	return head
}

func (hs *headSaver) Chain(hash common.Hash) *eth.Head {
	return hs.heads.HeadByHash(hash)
}

var _ httypes.HeadSaver = &NullSaver{}

type NullSaver struct{}

func (*NullSaver) Save(ctx context.Context, head *eth.Head) error          { return nil }
func (*NullSaver) LoadFromDB(ctx context.Context) (*eth.Head, error)       { return nil, nil }
func (*NullSaver) LatestHeadFromDB(ctx context.Context) (*eth.Head, error) { return nil, nil }
func (*NullSaver) LatestChain() *eth.Head                                  { return nil }
func (*NullSaver) Chain(hash common.Hash) *eth.Head                        { return nil }
