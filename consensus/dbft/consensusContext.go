package dbft

import (
	. "GoOnchain/common"
	"GoOnchain/crypto"
	tx "GoOnchain/core/transaction"
	 "GoOnchain/core/ledger"
	pl "GoOnchain/net/payload"
	ser "GoOnchain/common/serialization"
	cl "GoOnchain/client"
)

const ContextVersion uint32 = 0

type ConsensusContext struct {

	State ConsensusState
	PrevHash Uint256
	Height uint32
	ViewNumber byte
	Miners []*crypto.PubKey
	MinerIndex int
	PrimaryIndex uint32
	Timestamp uint32
	Nonce uint64
	NextMiner Uint160
	TransactionHashes []Uint256
	Transactions map[Uint256]*tx.Transaction
	Signatures [][]byte
	ExpectedView []byte

	txlist []*tx.Transaction

	header *ledger.Block
}

func (cxt *ConsensusContext)  M() int {
	return len(cxt.Miners) - (len(cxt.Miners) - 1) / 3
}

func NewConsensusContext() *ConsensusContext {
	return  &ConsensusContext{
	}
}

func (cxt *ConsensusContext)  ChangeView(viewNum byte)  {
	p := (cxt.Height - uint32(viewNum)) % uint32(len(cxt.Miners))
	cxt.State &= SignatureSent
	cxt.ViewNumber = viewNum
	if p >= 0 {
		cxt.PrimaryIndex = uint32(p)
	} else {
		cxt.PrimaryIndex = uint32(p) + uint32(len(cxt.Miners))
	}

	if cxt.State == Initial{
		cxt.TransactionHashes = nil
		cxt.Signatures = make([][]byte,len(cxt.Miners))
	}
	cxt.header = nil
}

func (cxt *ConsensusContext)  HasTxHash(txHash Uint256) bool {
	for _, hash :=  range cxt.TransactionHashes{
		if hash == txHash {
			return true
		}
	}
	return false
}

func (cxt *ConsensusContext)  MakeChangeView() *pl.ConsensusPayload {
	cv := &ChangeView{
		NewViewNumber: cxt.ExpectedView[cxt.MinerIndex],
	}
	return cxt.MakePayload(cv)
}

func (cxt *ConsensusContext)  MakeHeader() *ledger.Block {
	if cxt.TransactionHashes == nil {
		return nil
	}

	if cxt.header == nil{
		blockData := &ledger.Blockdata{
			Version: ContextVersion,
			PrevBlockHash: cxt.PrevHash,
			TransactionsRoot: crypto.ComputeRoot(cxt.TransactionHashes),
			Timestamp: cxt.Timestamp,
			Height: cxt.Height,
			ConsensusData: cxt.Nonce,
			NextMiner: cxt.NextMiner,
		}
		cxt.header = &ledger.Block{
			Blockdata: blockData,
			Transcations: []*tx.Transaction{},
		}
	}
	return cxt.header
}

func (cxt *ConsensusContext)  MakePayload(message ConsensusMessage) *pl.ConsensusPayload{
	message.ConsensusMessageData().ViewNumber = cxt.ViewNumber
	return &pl.ConsensusPayload{
		Version: ContextVersion,
		PrevHash: cxt.PrevHash,
		Height: cxt.Height,
		MinerIndex: uint16(cxt.MinerIndex),
		Timestamp: cxt.Timestamp,
		Data: ser.ToArray(message),
	}
}

func (cxt *ConsensusContext)  MakePerpareRequest() *pl.ConsensusPayload{
	preReq := &PrepareRequest{
		Nonce: cxt.Nonce,
		NextMiner: cxt.NextMiner,
		TransactionHashes: cxt.TransactionHashes,
		//TODO: MinerTransaction:
		Signature: cxt.Signatures[cxt.MinerIndex],
	}
	return cxt.MakePayload(preReq	)
}

func (cxt *ConsensusContext)  MakePerpareResponse(signature []byte) *pl.ConsensusPayload{
	preRes := &PrepareResponse{
		Signature: signature,
	}
	return cxt.MakePayload(preRes)
}

func (cxt *ConsensusContext)  GetSignaturesCount() (count int){
	count = 0
	for _,sig := range cxt.Signatures {
		if sig != nil {
			count += 1
		}
	}
	return count
}

func (cxt *ConsensusContext)  GetTransactionList()  []*tx.Transaction{
	if cxt.txlist == nil{
		cxt.txlist = []*tx.Transaction{}
		for _,TX := range cxt.Transactions {
			cxt.txlist = append(cxt.txlist,TX)
		}
	}
	return cxt.txlist
}

func (cxt *ConsensusContext)  GetTXByHashes()  []*tx.Transaction{
	TXs := []*tx.Transaction{}
	for _,hash := range cxt.TransactionHashes {
		if TX,ok:=cxt.Transactions[hash]; ok{
			TXs = append(TXs,TX)
		}
	}
	return TXs
}

func (cxt *ConsensusContext)  CheckTxHashesExist() bool {
	for _,hash := range cxt.TransactionHashes {
		if _,ok:=cxt.Transactions[hash]; !ok{
			return false
		}
	}
	return true
}

func (cxt *ConsensusContext) Reset(client *cl.Client){
	cxt.State = Initial
	cxt.PrevHash = ledger.DefaultLedger.Blockchain.CurrentBlockHash()
	cxt.Height = ledger.DefaultLedger.Blockchain.BlockHeight + 1
	cxt.ViewNumber = 0
	cxt.Miners = ledger.DefaultLedger.Blockchain.GetMiners()
	cxt.MinerIndex = -1

	minerLen := len(cxt.Miners)
	cxt.PrimaryIndex = cxt.Height % uint32(minerLen)
	cxt.TransactionHashes = nil
	cxt.Signatures = make([][]byte,minerLen)
	cxt.ExpectedView = make([]byte,minerLen)

	for i:=0;i<minerLen ;i++  {
		if client.ContainsAccount(cxt.Miners[i]){
			cxt.MinerIndex = i
			break
		}
	}
	cxt.header = nil
}