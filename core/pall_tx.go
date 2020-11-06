package core

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"sync"
	"time"
)

type pallTxManager struct {
	block *types.Block
	txLen int
	bc    *BlockChain

	mubase         sync.RWMutex
	baseStateDB    *state.StateDB
	mergedReceipts map[int]*types.Receipt
	mergedRW       map[int]map[common.Address]bool
	ch             chan struct{}
	//mergedNumber   int

	lastHandleInGroup map[int]int
	txIndexToGroupID  map[int]int

	groupList map[int][]int

	txQueue      chan int
	receiptQueue []*ReceiptWithIndex

	gp uint64

	ended bool
	//signer types.Signer
}

type ReceiptWithIndex struct {
	st      *state.StateDB
	txIndex int
	receipt *types.Receipt
}

func NewPallTxManage(block *types.Block, st *state.StateDB, bc *BlockChain) *pallTxManager {
	st.MergedIndex = -1
	txLen := len(block.Transactions())
	p := &pallTxManager{
		block:             block,
		txLen:             txLen,
		baseStateDB:       st,
		bc:                bc,
		mergedReceipts:    make(map[int]*types.Receipt, 0),
		mergedRW:          make(map[int]map[common.Address]bool),
		ch:                make(chan struct{}, 1),
		txQueue:           make(chan int, txLen),
		txIndexToGroupID:  make(map[int]int, 0),
		lastHandleInGroup: make(map[int]int),

		//mergedNumber: -1,
		groupList:    make(map[int][]int, 0),
		receiptQueue: make([]*ReceiptWithIndex, txLen, txLen),

		gp: block.GasLimit(),
	}

	signer := types.MakeSigner(bc.chainConfig, block.Number())
	addrToGroupID := make(map[common.Address]int, 0)
	for index, tx := range block.Transactions() {
		sender, _ := types.Sender(signer, tx)
		groupID := p.calGroup(addrToGroupID, sender, tx.To())

		p.groupList[groupID] = append(p.groupList[groupID], index)
		p.txIndexToGroupID[index] = groupID
	}

	for index := 0; index < 8; index++ {
		go p.txLoop()
	}

	for index := 0; index < len(p.groupList); index++ {
		p.AddTxToQueue(p.groupList[index][0])
	}
	fmt.Println("block---ready to pall", block.NumberU64(), p.groupList)
	return p
}

func (p *pallTxManager) calGroup(mp map[common.Address]int, from common.Address, to *common.Address) int {
	groupID := len(p.groupList)

	if to != nil {
		if data, ok := mp[*to]; ok {
			groupID = data
		}
	}
	if data, ok := mp[from]; ok {
		groupID = data
	}

	mp[from] = groupID
	if to != nil {
		mp[*to] = groupID
	}
	return groupID
}

func (p *pallTxManager) AddTxToQueue(txIndex int) {
	p.txQueue <- txIndex
}

func (p *pallTxManager) GetTxFromQueue() (int, bool) {
	data, ok := <-p.txQueue
	return data, ok == false
}

func (p *pallTxManager) AddReceiptToQueue(re *ReceiptWithIndex) {
	p.receiptQueue[re.txIndex] = re
	startTxIndex := re.txIndex

	p.mubase.Lock()
	defer p.mubase.Unlock()

	if p.ended {
		return
	}

	for p.baseStateDB.MergedIndex+1 == startTxIndex && startTxIndex < p.txLen && p.receiptQueue[startTxIndex] != nil {
		p.handleReceipt(p.receiptQueue[startTxIndex])
		startTxIndex++
	}

	if p.baseStateDB.MergedIndex+1 == p.txLen {
		p.ch <- struct{}{}
		p.ended = true

		//fmt.Println("========ssssscccccccfffffffffff", p.mergedRW)
		//fmt.Println("========ssssscccccccfffffffffff", p.baseStateDB.GetObjs())
		//p.baseStateDB.Prepare(common.Hash{},common.Hash{},)
		p.baseStateDB.ENd(p.mergedRW, p.txLen)

	}
}

func (p *pallTxManager) txLoop() {
	for {
		tx, isClosed := p.GetTxFromQueue()
		if isClosed {
			return
		}
		if !p.handleTx(tx) && !p.ended {
			p.AddTxToQueue(tx)
		}
	}
}

func (p *pallTxManager) handleReceipt(rr *ReceiptWithIndex) {
	if rr.st.CanMerge(p.baseStateDB, p.mergedRW, p.block.Coinbase()) {
		rr.st.Merge(p.baseStateDB, p.block.Coinbase())
		p.gp -= rr.receipt.GasUsed
		p.mergedReceipts[rr.txIndex] = rr.receipt
		p.mergedRW[rr.txIndex] = rr.st.ThisTxRW
		p.baseStateDB.MergedIndex = rr.txIndex

		groupID := p.txIndexToGroupID[rr.txIndex]
		p.lastHandleInGroup[groupID]++
		if p.lastHandleInGroup[groupID] < len(p.groupList[groupID]) {
			p.AddTxToQueue(p.groupList[groupID][p.lastHandleInGroup[groupID]])
		}
		p.baseStateDB.Print(fmt.Sprintf("blockNumber=%v merged end mergedNumbe=%v", p.block.NumberU64(), p.baseStateDB.MergedIndex))

	} else {
		//fmt.Println("cccccccccccccc", p.block.NumberU64(), p.baseStateDB.MergedIndex, rr.txIndex)
		p.AddTxToQueue(rr.txIndex)
	}
}

func (p *pallTxManager) handleTx(txIndex int) bool {
	tx := p.block.Transactions()[txIndex]
	p.mubase.Lock()
	if txIndex <= p.baseStateDB.MergedIndex || p.receiptQueue[txIndex] != nil || p.ended {
		//fmt.Println("ddddddddddd", txIndex <= p.baseStateDB.MergedIndex, p.receiptQueue[txIndex] != nil, p.ended)
		p.mubase.Unlock()
		return true
	}
	preBaseMerged := p.baseStateDB.MergedIndex
	if p.baseStateDB.MergedIndex != p.baseStateDB.MergedIndex {
		panic(fmt.Errorf("bug here %v != %v", p.baseStateDB.MergedIndex, p.baseStateDB.MergedIndex))
	}
	st, err := state.New(common.Hash{}, p.bc.stateCache, p.bc.snaps)
	if err != nil {
		panic(err)
	}
	st.Scf = p.baseStateDB.Scf
	st.MergedIndex = p.baseStateDB.MergedIndex
	gas := p.gp
	p.mubase.Unlock()

	st.Print(fmt.Sprintf("blockNumber=%v apply tx before preBaseMerged=%v txIndex=%v", p.block.NumberU64(), preBaseMerged, txIndex))

	st.Prepare(tx.Hash(), p.block.Hash(), txIndex)

	//fmt.Println("TTTTTTTTTTTTTTTTTTTTTT", p.block.NumberU64(), txIndex, tx.Nonce(), tx.Hash().String())
	receipt, err := ApplyTransaction(p.bc.chainConfig, p.bc, nil, new(GasPool).AddGas(gas), st, p.block.Header(), tx, nil, p.bc.vmConfig)
	st.Print(fmt.Sprintf("blockNumber=%v apply tx end preBaseMerged=%v txIndex=%v ermsg=%v", p.block.NumberU64(), preBaseMerged, txIndex, err))
	if err != nil {
		fmt.Println("---apply tx err---", err, "blockNumber", p.block.NumberU64(), "baseMergedNumber", preBaseMerged, "currTxIndex", txIndex, "groupList", p.groupList)
		time.Sleep(5 * time.Second)
		return false
	}

	//fmt.Println("ready to add receipt", p.block.NumberU64(), preBaseMerged, txIndex)
	p.AddReceiptToQueue(&ReceiptWithIndex{
		st:      st,
		txIndex: txIndex,
		receipt: receipt,
	})
	//fmt.Println("end to add receipt", p.block.NumberU64(), preBaseMerged, txIndex)
	return true

}

func (p *pallTxManager) GetReceiptsAndLogs() (types.Receipts, []*types.Log, uint64) {
	receipts := make(types.Receipts, 0)
	logs := make([]*types.Log, 0)

	CumulativeGasUsed := uint64(0)
	for index := 0; index < p.txLen; index++ {
		CumulativeGasUsed = CumulativeGasUsed + p.mergedReceipts[index].GasUsed
		p.mergedReceipts[index].CumulativeGasUsed = CumulativeGasUsed
		//p.mergedReceipts[index].Bloom = types.CreateBloom(types.Receipts{p.mergedReceipts[index]})

		receipts = append(receipts, p.mergedReceipts[index])
		logs = append(logs, p.mergedReceipts[index].Logs...)
	}
	return receipts, logs, CumulativeGasUsed
}
