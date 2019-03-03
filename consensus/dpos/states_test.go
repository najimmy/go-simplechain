package dpos

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/nacamp/go-simplechain/rlp"
	"github.com/nacamp/go-simplechain/tests"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/storage"
	"github.com/stretchr/testify/assert"
)

func candidate(state *DposState, address common.Address) (balance *big.Int) {
	encodedBytes, _ := state.Candidate.Get(address[:])
	balance = new(big.Int)
	_ = rlp.Decode(bytes.NewReader(encodedBytes), balance)
	return balance
}

func TestStakeUnstake(t *testing.T) {
	_storage, _ := storage.NewMemoryStorage()
	state, err := NewInitState(common.Hash{}, 0, _storage)
	assert.NoError(t, err)
	_ = state

	//var newAddr = "0x1df75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2"
	err = state.Stake(common.HexToAddress(tests.Addr0), common.HexToAddress(tests.Addr0), new(big.Int).SetUint64(0))
	assert.Error(t, err)
	err = state.Stake(common.HexToAddress(tests.Addr0), common.HexToAddress(tests.Addr0), new(big.Int).SetUint64(10))
	assert.NoError(t, err)
	state.Stake(common.HexToAddress(tests.Addr0), common.HexToAddress(tests.Addr2), new(big.Int).SetUint64(20))
	state.Stake(common.HexToAddress(tests.Addr1), common.HexToAddress(tests.Addr2), new(big.Int).SetUint64(30))
	assert.True(t, candidate(state, tests.Address2).Cmp(new(big.Int).SetUint64(50)) == 0)

	err = state.Unstake(tests.Address2, tests.Address2, new(big.Int).SetUint64(10))
	assert.NoError(t, err)
	err = state.Unstake(tests.Address2, tests.Address2, new(big.Int).SetUint64(50))
	assert.Error(t, err)
}

func TestGetNewElectedTime(t *testing.T) {
	assert.Equal(t, uint64(0), GetNewElectedTime(0, 26, 3, 3, 3))
	assert.Equal(t, uint64(27), GetNewElectedTime(0, 27, 3, 3, 3))
}

/*
func (ds *DposState) GetMiners(minerHash common.Hash) ([]common.Address, error) {
	miner := []common.Address{}
	decodedBytes, _ := ds.Miner.Get(minerHash[:])
	rlp.NewStream(bytes.NewReader(decodedBytes), 0).Decode(&miner)
	return miner, nil
}

func (ds *DposState) GetNewRoundMiners(electedTime uint64, totalMiners int) ([]common.Address, error) {
	iter, err := ds.Candidate.Iterator(nil)
	if err != nil {
		return nil, err
	}
	exist, _ := iter.Next()
	candidates := []core.BasicAccount{}
	for exist {
		account := core.BasicAccount{Address: common.Address{}}

		// encodedBytes1 := iter.Key()
		// key := new([]byte)
		// rlp.NewStream(bytes.NewReader(encodedBytes1), 0).Decode(key)
		account.Address = common.BytesToAddress(iter.Key())

		encodedBytes2 := iter.Value()
		value := new(big.Int)
		rlp.NewStream(bytes.NewReader(encodedBytes2), 0).Decode(value)
		account.Balance = value

		candidates = append(candidates, account)
		exist, err = iter.Next()
	}

	if len(candidates) < totalMiners {
		return nil, errors.New("The number of candidated miner is smaller than the minimum miner number.")
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Balance.Cmp(candidates[j].Balance) > 0
	})

	candidates = candidates[:totalMiners]
	candidateAddrs := []common.Address{}
	for _, v := range candidates {
		candidateAddrs = append(candidateAddrs, v.Address)
	}
	shuffle(candidateAddrs, int64(electedTime))
	return candidateAddrs, nil
}

func (ds *DposState) PutMiners(miners []common.Address) (hash common.Hash, err error) {
	encodedBytes, err := rlp.EncodeToBytes(miners)
	if err != nil {
		return common.Hash{}, err
	}
	hashBytes := crypto.Sha3b256(encodedBytes)
	ds.Miner.Put(hashBytes, encodedBytes)
	copy(hash[:], hashBytes)
	return hash, nil
}

func (ds *DposState) Put(blockNumber, electedTime uint64, minersHash common.Hash) error {
	vals := make([]byte, 0)
	keyEncodedBytes, err := rlp.EncodeToBytes(blockNumber)
	if err != nil {
		return err
	}
	encodedTimeBytes, err := rlp.EncodeToBytes(electedTime)
	if err != nil {
		return err
	}

	vals = append(vals, ds.Candidate.RootHash()...)
	vals = append(vals, ds.Voter.RootHash()...)
	vals = append(vals, minersHash[:]...)
	vals = append(vals, encodedTimeBytes...)
	_, err = ds.Miner.Put(crypto.Sha3b256(keyEncodedBytes), vals)
	if err != nil {
		return err
	}

	return nil
}


func (ds *DposState) Get(blockNumber uint64) (common.Hash, common.Hash, common.Hash, uint64, error) {
	keyEncodedBytes, err := rlp.EncodeToBytes(blockNumber)
	if err != nil {
		return common.Hash{}, common.Hash{}, common.Hash{}, 0, err
	}
	//TODO: check minimum key size
	encbytes, err := ds.Miner.Get(crypto.Sha3b256(keyEncodedBytes))
	if err != nil {
		return common.Hash{}, common.Hash{}, common.Hash{}, 0, err
	}
	if len(encbytes) < common.HashLength*3 {
		return common.Hash{}, common.Hash{}, common.Hash{}, 0, errors.New("Bytes lenght must be more than 64 bits")
	}

	electedTime := uint64(0)
	err = rlp.Decode(bytes.NewReader(encbytes[common.HashLength*3:]), &electedTime)
	if err != nil {
		return common.Hash{}, common.Hash{}, common.Hash{}, 0, err
	}
	return common.BytesToHash(encbytes[:common.HashLength]),
		common.BytesToHash(encbytes[common.HashLength : common.HashLength*2]),
		common.BytesToHash(encbytes[common.HashLength*2 : common.HashLength*3]),
		electedTime, nil
}

func (ds *DposState) RootHash() (hash common.Hash) {
	copy(hash[:], ds.Miner.RootHash())
	return hash
}

func (ds *DposState) Clone() (core.ConsensusState, error) {
	tr1, err1 := ds.Candidate.Clone()
	if err1 != nil {
		return nil, err1
	}
	tr2, err2 := ds.Miner.Clone()
	if err2 != nil {
		return nil, err2
	}
	tr3, err3 := ds.Voter.Clone()
	if err3 != nil {
		return nil, err3
	}
	return &DposState{
		Candidate:   tr1,
		Miner:       tr2,
		Voter:       tr3,
		MinersHash:  ds.MinersHash,
		ElectedTime: ds.ElectedTime,
	}, nil
}

func (cs *DposState) ExecuteTransaction(block *core.Block, txIndex int, account *core.Account) (err error) {

	tx := block.Transactions[txIndex]
	amount := new(big.Int)
	err = rlp.Decode(bytes.NewReader(tx.Payload.Data), amount)
	if err != nil {
		return err
	}
	if tx.Payload.Code == core.TxCVoteStake {
		err = account.Stake(tx.To, amount)
		if err != nil {
			return err
		}
		return cs.Stake(account.Address, tx.To, amount)
	} else if tx.Payload.Code == core.TxCVoteUnStake {
		err = account.UnStake(tx.To, amount)
		if err != nil {
			return err
		}
		return cs.Unstake(account.Address, tx.To, amount)
	}
	return nil
}





func shuffle(slice []common.Address, seed int64) {
	r := rand.New(rand.NewSource(seed))
	for len(slice) > 0 {
		n := len(slice)
		randIndex := r.Intn(n)
		slice[n-1], slice[randIndex] = slice[randIndex], slice[n-1]
		slice = slice[:n-1]
	}
}

func (bc *BlockChain) PutMinerState(block *Block) error {

	// save status
	ms := block.MinerState
	minerGroup, voterBlock, err := ms.GetMinerGroup(bc, block)
	if err != nil {
		return err
	}
	//TODO: we need to test  when voter transaction make
	//make new miner group
	if voterBlock.Header.Height == block.Header.Height {

		ms.Put(minerGroup, block.Header.VoterHash) //TODO voterhash
	}
	//else use parent miner group
	//TODO: check after 3 seconds(block creation) and 3 seconds(mining order)
	index := (block.Header.Time % 9) / 3
	if minerGroup[index] != block.Header.Coinbase {
		return errors.New("minerGroup[index] != block.Header.Coinbase")
	}

	return nil

}

func (ds *DposState) GetMiners(minerHash common.Hash) ([]common.Address, error) {
	// encodedBytes1, err := rlp.EncodeToBytes(electedTime)
	// if err != nil {
	// 	return nil, err
	// }
	miner := []common.Address{}
	decodedBytes, _ := ds.Miner.Get(minerHash[:])
	rlp.NewStream(bytes.NewReader(decodedBytes), 0).Decode(&miner)
	return miner, nil
}

*/
