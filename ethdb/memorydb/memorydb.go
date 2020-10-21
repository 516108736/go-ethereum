// Copyright 2018 The go-ethereum Authors
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

// Package memorydb implements the key-value database layer based on memory maps.
package memorydb

import (
	"encoding/binary"
	"errors"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ledgerwatch/turbo-geth/common/dbutils"
	tethdb "github.com/ledgerwatch/turbo-geth/ethdb"
)

var (
	// errMemorydbClosed is returned if a memory database was already closed at the
	// invocation of a data access operation.
	errMemorydbClosed = errors.New("database closed")

	// errMemorydbNotFound is returned if a key is requested that is not found in
	// the provided memory database.
	errMemorydbNotFound = errors.New("not found")
	bucket              = "scf"
)

// Database is an ephemeral key-value store. Apart from basic data storage
// functionality it also supports batch writes and iterating over the keyspace in
// binary-alphabetical order.
type Database struct {
	lmdb tethdb.Database
}

func uint64ToBytes(u uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, u)
	return b
}

func (d *Database) HasAncient(kind string, number uint64) (bool, error) {
	return d.lmdb.Has(kind, uint64ToBytes(number))
}

func (d *Database) Ancient(kind string, number uint64) ([]byte, error) {
	return d.lmdb.Get(kind, uint64ToBytes(number))
}

func (d *Database) Ancients() (uint64, error) {
	panic("implement me")
}

func (d *Database) AncientSize(kind string) (uint64, error) {
	panic("implement me")
}

func (d *Database) AppendAncient(number uint64, hash, header, body, receipt, td []byte) error {
	panic("implement me")
}

func (d *Database) TruncateAncients(n uint64) error {
	panic("implement me")
}

func (d *Database) Sync() error {
	panic("implement me")
}

// New returns a wrapped map with all the required database interface methods
// implemented.
func New() *Database {
	return &Database{}
}

func (d *Database) SetPath(path string) {
	kv, err := tethdb.NewLMDB().Path(path).WithBucketsConfig(func(defaultBuckets dbutils.BucketsCfg) dbutils.BucketsCfg {
		return map[string]dbutils.BucketConfigItem{
			bucket: {
				Flags: 0,
			},
		}
	}).Open()
	if err != nil {
		panic(err)
	}
	db := tethdb.NewObjectDatabase(kv)
	d.lmdb = db
}

// NewWithCap returns a wrapped map pre-allocated to the provided capcity with
// all the required database interface methods implemented.
func NewWithCap(size int) *Database {
	panic("not 1")
}

// Close deallocates the internal map and ensures any consecutive data access op
// failes with an error.
func (db *Database) Close() error {
	db.lmdb.Close()
	return nil
}

// Has retrieves if a key is present in the key-value store.
func (db *Database) Has(key []byte) (bool, error) {
	return db.lmdb.Has(bucket, key)
}

// Get retrieves the given key if it's present in the key-value store.
func (db *Database) Get(key []byte) ([]byte, error) {
	return db.lmdb.Get(bucket, key)
}

// Put inserts the given value into the key-value store.
func (db *Database) Put(key []byte, value []byte) error {
	return db.lmdb.Put(bucket, key, value)
}

// Delete removes the key from the key-value store.
func (db *Database) Delete(key []byte) error {
	return db.lmdb.Delete(bucket, key)
}

// NewBatch creates a write-only key-value store that buffers changes to its host
// database until a final write is called.
func (db *Database) NewBatch() ethdb.Batch {
	b := db.lmdb.NewBatch()
	return &batch{
		batch: b,
	}
}

// NewIterator creates a binary-alphabetical iterator over a subset
// of database content with a particular key prefix, starting at a particular
// initial key (or after, if it does not exist).
func (db *Database) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	panic("not support NewIterator")
}

// Stat returns a particular internal stat of the database.
func (db *Database) Stat(property string) (string, error) {
	panic("not support Stat ")
}

// Compact is not supported on a memory database, but there's no need either as
// a memory database doesn't waste space anyway.
func (db *Database) Compact(start []byte, limit []byte) error {
	return nil
}

// Len returns the number of entries currently present in the memory database.
//
// Note, this method is only used for testing (i.e. not public in general) and
// does not have explicit checks for closed-ness to allow simpler testing code.
func (db *Database) Len() int {
	ks, err := db.lmdb.Keys()
	if err != nil {
		panic(err)
	}
	return len(ks)
}

// keyvalue is a key-value tuple tagged with a deletion field to allow creating
// memory-database write batches.
type keyvalue struct {
	key    []byte
	value  []byte
	delete bool
}

// batch is a write-only memory batch that commits changes to its host
// database when Write is called. A batch cannot be used concurrently.
type batch struct {
	batch tethdb.DbWithPendingMutations
}

// Put inserts the given value into the batch for later committing.
func (b *batch) Put(key, value []byte) error {
	return b.batch.Put(bucket, key, value)
}

// Delete inserts the a key removal into the batch for later committing.
func (b *batch) Delete(key []byte) error {
	return b.batch.Delete(bucket, key)
}

// ValueSize retrieves the amount of data queued up for writing.
func (b *batch) ValueSize() int {
	return b.batch.BatchSize()
}

// Write flushes any accumulated data to the memory database.
func (b *batch) Write() error {
	_, err := b.batch.Commit()
	return err
}

// Reset resets the batch for reuse.
func (b *batch) Reset() {
	b.batch.Rollback()
}

// Replay replays the batch contents.
func (b *batch) Replay(w ethdb.KeyValueWriter) error {
	return nil
}
