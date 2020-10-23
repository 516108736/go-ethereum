// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"fmt"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/leveldb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"io/ioutil"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Tests that transactions can be added to strict lists and list contents and
// nonce boundaries are correctly maintained.
func TestStrictTxListAdd(t *testing.T) {
	// Generate a list of transactions to insert
	key, _ := crypto.GenerateKey()

	txs := make(types.Transactions, 1024)
	for i := 0; i < len(txs); i++ {
		txs[i] = transaction(uint64(i), 0, key)
	}
	// Insert the transactions in a random order
	list := newTxList(true)
	for _, v := range rand.Perm(len(txs)) {
		list.Add(txs[v], DefaultTxPoolConfig.PriceBump)
	}
	// Verify internal state
	if len(list.txs.items) != len(txs) {
		t.Errorf("transaction count mismatch: have %d, want %d", len(list.txs.items), len(txs))
	}
	for i, tx := range txs {
		if list.txs.items[tx.Nonce()] != tx {
			t.Errorf("item %d: transaction mismatch: have %v, want %v", i, list.txs.items[tx.Nonce()], tx)
		}
	}
}

func BenchmarkTxListAdd(t *testing.B) {
	// Generate a list of transactions to insert
	key, _ := crypto.GenerateKey()

	txs := make(types.Transactions, 100000)
	for i := 0; i < len(txs); i++ {
		txs[i] = transaction(uint64(i), 0, key)
	}
	// Insert the transactions in a random order
	list := newTxList(true)
	priceLimit := big.NewInt(int64(DefaultTxPoolConfig.PriceLimit))
	t.ResetTimer()
	for _, v := range rand.Perm(len(txs)) {
		list.Add(txs[v], DefaultTxPoolConfig.PriceBump)
		list.Filter(priceLimit, DefaultTxPoolConfig.PriceBump)
	}
}

func GetEthDB(lmdb bool) ethdb.Database {
	dir, err := ioutil.TempDir("", "disklayer-test")
	if err != nil {
		panic(err)
	}
	fmt.Println("file", dir, lmdb)
	var kvdb ethdb.KeyValueStore
	if lmdb {
		kk := memorydb.New()
		kk.SetPath(dir)
		kvdb = kk
	} else {
		kvdb, err = leveldb.New(dir, 512, 524288, "")
		if err != nil {
			panic(err)
		}
	}

	frdb, err := rawdb.NewDatabaseWithFreezer(kvdb, "s", "")
	if err != nil {
		panic(err)
	}
	return frdb

}

func makeData(number int) [][]byte {
	ans := make([][]byte, 0)
	for index := 1; index <= number; index++ {
		ans = append(ans, new(big.Int).SetUint64(uint64(index)).Bytes())
	}
	return ans
}

func TestAsd1(t *testing.T) {
	numbers := 10
	datas := makeData(numbers)

	db := GetEthDB(false)
	ts := time.Now()
	for index := 0; index < numbers; index++ {
		if err := db.Put(datas[index], datas[index]); err != nil {
			panic(err)
		}
	}

	a, err := db.Get(new(big.Int).SetUint64(8).Bytes())
	fmt.Println("end leveldb", time.Now().Sub(ts).Seconds(), a, err)

	dblmdb := GetEthDB(true)
	ts = time.Now()
	for index := 0; index < numbers; index++ {
		fmt.Println("err", err, index, string(datas[index]), dblmdb)
		if err := dblmdb.Put(datas[index], datas[index]); err != nil {

			panic(err)
		}
	}

	a, err = dblmdb.Get(new(big.Int).SetUint64(8).Bytes())
	fmt.Println("end lmdb", time.Now().Sub(ts).Seconds(), string(a), err)
}
