// Package btree provide a disk-based btree
package btree

import (
	"encoding/binary"
	"io"
	"math/rand"
	"os"
	"unsafe"

	"github.com/pkg/errors"
)

const (
	sha1Size        = 20 // md5 sha1
	tableSize       = (4096 - 1) / int(unsafe.Sizeof(item{}))
	cacheSlots      = 23 // prime
	superSize       = unsafe.Sizeof(super{})
	tableStructSize = int(unsafe.Sizeof(table{}))
)

type item struct {
	sha1   [sha1Size]byte
	offset int64
	child  int64
}

type table struct {
	items [tableSize]item
	size  int
}

type cache struct {
	table  *table
	offset int64
}

type super struct {
	top     int64
	freeTop int64
	alloc   int64
}

// Btree ...
type Btree struct {
	fd      *os.File
	top     int64
	freeTop int64
	alloc   int64
	cache   [23]cache

	inAllocator  bool
	deleteLarger bool
}

func (bt *Btree) get(offset int64) *table {
	assert(offset != 0)

	// take from cache
	slot := &bt.cache[offset%cacheSlots]
	if slot.offset == offset {
		return slot.table
	}

	table := new(table)

	bt.fd.Seek(offset, io.SeekStart)
	err := binary.Read(bt.fd, binary.LittleEndian, table) // todo(wdvxdr): efficient reading
	if err != nil {
		panic(errors.Wrap(err, "btree I/O error"))
	}
	return table
}

func (bt *Btree) put(t *table, offset int64) {
	assert(offset != 0)

	/* overwrite cache */
	slot := &bt.cache[offset%cacheSlots]
	slot.table = t
	slot.offset = offset
}

func (bt *Btree) flush(t *table, offset int64) {
	assert(offset != 0)

	bt.fd.Seek(offset, io.SeekStart)
	err := binary.Write(bt.fd, binary.LittleEndian, t)
	if err != nil {
		panic(errors.Wrap(err, "btree I/O error"))
	}
	bt.put(t, offset)
}

func (bt *Btree) flushSuper() {
	bt.fd.Seek(0, io.SeekStart)
	super := super{
		top:     bt.top,
		freeTop: bt.freeTop,
		alloc:   bt.alloc,
	}
	err := binary.Write(bt.fd, binary.LittleEndian, super)
	if err != nil {
		panic(errors.Wrap(err, "btree I/O error"))
	}
}

// Open opens an existed btree file
func Open(name string) (*Btree, error) {
	btree := new(Btree)
	fd, err := os.OpenFile(name, os.O_RDWR, 0o644)
	if err != nil {
		return nil, errors.Wrap(err, "btree open file failed")
	}
	btree.fd = fd

	super := super{}
	err = binary.Read(fd, binary.LittleEndian, &super)
	btree.top = super.top
	btree.freeTop = super.freeTop
	btree.alloc = super.alloc
	return btree, errors.Wrap(err, "btree read meta info failed")
}

// Create creates a database
func Create(name string) (*Btree, error) {
	btree := new(Btree)
	fd, err := os.OpenFile(name, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0o644)
	if err != nil {
		return nil, errors.Wrap(err, "btree open file failed")
	}

	btree.fd = fd
	btree.alloc = int64(superSize)
	btree.flushSuper()
	return btree, nil
}

// Close closes the database
func (bt *Btree) Close() error {
	err := bt.Close()
	for i := 0; i < cacheSlots; i++ {
		bt.cache[i] = cache{}
	}
	return errors.Wrap(err, "btree close failed")
}

func collapse(bt *Btree, offset int64) int64 {
	table := bt.get(offset)
	if table.size != 0 {
		/* unable to collapse */
		bt.put(table, offset)
		return offset
	}
	ret := table.items[0].child
	bt.put(table, offset)

	/*
	 * WARNING: this is dangerous as the chunk is added to allocation tree
	 * before the references to it are removed!
	 */
	bt.freeChunk(offset, int(unsafe.Sizeof(table)))
	return ret
}

// split a table. The pivot item is stored to 'sha1' and 'offset'.
// Returns offset to the new table.
func (bt *Btree) split(t *table, hash *byte, offset *int64) int64 {
	copysha1(hash, &t.items[tableSize/2].sha1[0])
	*offset = t.items[tableSize/2].offset

	ntable := new(table)
	ntable.size = t.size - tableSize/2 - 1

	t.size = tableSize / 2

	copy(ntable.items[:ntable.size+1], t.items[tableSize/2+1:])

	noff := bt.allocChunk(tableStructSize)
	bt.flush(ntable, noff)

	// make sure data is written before a reference is added to it
	_ = bt.fd.Sync()
	return noff
}

// takeSmallest find and remove the smallest item from the given table. The key of the item
// is stored to 'sha1'. Returns offset to the item
func (bt *Btree) takeSmallest(toff int64, sha1 *byte) int64 {
	table := bt.get(toff)
	assert(table.size > 0)

	var off int64
	child := table.items[0].child
	if child == 0 {
		off = bt.remove(table, 0, sha1)
	} else {
		/* recursion */
		off = bt.takeSmallest(child, sha1)
		table.items[0].child = collapse(bt, child)
	}
	bt.flush(table, toff)

	// make sure data is written before a reference is added to it
	_ = bt.fd.Sync()
	return off
}

// takeLargest find and remove the largest item from the given table. The key of the item
// is stored to 'sha1'. Returns offset to the item
func (bt *Btree) takeLargest(toff int64, sha1 *byte) int64 {
	table := bt.get(toff)
	assert(table.size > 0)

	var off int64
	child := table.items[table.size].child
	if child == 0 {
		off = bt.remove(table, table.size-1, sha1)
	} else {
		/* recursion */
		off = bt.takeLargest(child, sha1)
		table.items[table.size].child = collapse(bt, child)
	}
	bt.flush(table, toff)

	// make sure data is written before a reference is added to it
	_ = bt.fd.Sync()
	return off
}

// remove an item in position 'i' from the given table. The key of the
// removed item is stored to 'sha1'. Returns offset to the item.
func (bt *Btree) remove(t *table, i int, sha1 *byte) int64 {
	assert(i < t.size)

	if sha1 != nil {
		copysha1(sha1, &t.items[i].sha1[0])
	}

	offset := t.items[i].offset
	lc := t.items[i].child
	rc := t.items[i+1].child

	if lc != 0 && rc != 0 {
		/* replace the removed item by taking an item from one of the
		   child tables */
		var noff int64
		if rand.Int()&1 != 0 {
			noff = bt.takeLargest(lc, &t.items[i].sha1[0])
			t.items[i].child = collapse(bt, lc)
		} else {
			noff = bt.takeSmallest(rc, &t.items[i].sha1[0])
			t.items[i+1].child = collapse(bt, rc)
		}
		t.items[i].child = noff
	} else {
		// memmove(&table->items[i], &table->items[i + 1],
		//	(table->size - i) * sizeof(struct btree_item));
		// table->size--;
		for j := i; j < t.size-i; j++ { // fuck you, go!
			t.items[j] = t.items[j+1]
		}
		t.size--

		if lc != 0 {
			t.items[i].child = lc
		} else {
			t.items[i].child = rc
		}
	}
	return offset
}

func (bt *Btree) insert(toff int64, sha1 *byte, data []byte, len int) int64 {
	table := bt.get(toff)
	assert(table.size < tableSize-1)

	left, right := 0, table.size
	for left < right {
		mid := (right-left)>>1 + left
		switch cmp := cmp(sha1, &table.items[mid].sha1[0]); {
		case cmp == 0:
			// already in the table
			ret := table.items[mid].offset
			bt.put(table, toff)
			return ret
		case cmp < 0:
			right = mid
		default:
			left = mid + 1
		}
	}
	i := left

	var off, rc, ret int64
	lc := table.items[i].child
	if lc != 0 {
		/* recursion */
		ret = bt.insert(lc, sha1, data, len)

		/* check if we need to split */
		child := bt.get(lc)
		if child.size < tableSize-1 {
			/* nothing to do */
			bt.put(table, toff)
			bt.put(child, lc)
			return ret
		}
		/* overwrites SHA-1 */
		rc = bt.split(child, sha1, &off)
		/* flush just in case changes happened */
		bt.flush(child, lc)

		// make sure data is written before a reference is added to it
		_ = bt.fd.Sync()
	} else {
		off = bt.insertData(data, len)
		ret = off
	}

	table.size++
	// todo:
	// memmove(&table->items[i + 1], &table->items[i],
	// 	(table->size - i) * sizeof(struct btree_item));
	copysha1(&table.items[i].sha1[0], sha1)
	table.items[i].offset = off
	table.items[i].child = lc
	table.items[i+1].child = rc

	bt.flush(table, toff)
	return ret
}

func (bt *Btree) insertData(data []byte, size int) int64 {
	if data == nil {
		return int64(size)
	}
	assert(len(data) == size)

	offset := bt.allocChunk(4 + len(data))

	bt.fd.Seek(offset, io.SeekStart)
	err := binary.Write(bt.fd, binary.LittleEndian, int32(len(data)))
	if err != nil {
		panic(errors.Wrap(err, "btree I/O error"))
	}
	_, err = bt.fd.Write(data)
	if err != nil {
		panic(errors.Wrap(err, "btree I/O error"))
	}

	// make sure data is written before a reference is added to it
	_ = bt.fd.Sync()
	return offset
}

// delete remove an item with key 'sha1' from the given table. The offset to the
// removed item is returned.
// Please note that 'sha1' is overwritten when called inside the allocator.
func (bt *Btree) delete(offset int64, hash *byte) int64 {
	if offset == 0 {
		return 0
	}
	table := bt.get(offset)

	left, right := 0, table.size
	for left < right {
		i := (right-left)>>1 + left
		switch cmp := cmp(hash, &table.items[i].sha1[0]); {
		case cmp == 0:
			// found
			ret := bt.remove(table, i, hash)
			bt.flush(table, offset)
			return ret
		case cmp < 0:
			right = i
		default:
			left = i + 1
		}
	}

	// not found - recursion
	i := left
	child := table.items[i].child
	ret := bt.delete(child, hash)
	if ret != 0 {
		table.items[i].child = collapse(bt, child)
	}

	if ret == 0 && bt.deleteLarger && i < table.size {
		ret = bt.remove(table, i, hash)
	}
	if ret != 0 {
		/* flush just in case changes happened */
		bt.flush(table, offset)
	} else {
		bt.put(table, offset)
	}
	return ret
}

func (bt *Btree) insertTopLevel(toff *int64, sha1 *byte, data []byte, len int) int64 {
	var off, ret, rc int64
	if *toff != 0 {
		ret = bt.insert(*toff, sha1, data, len)

		/* check if we need to split */
		table := bt.get(*toff)
		if table.size < tableSize-1 {
			/* nothing to do */
			bt.put(table, *toff)
			return ret
		}
		rc = bt.split(table, sha1, &off)
		bt.flush(table, *toff)
	} else {
		off = bt.insertData(data, len)
		ret = off
	}

	/* create new top level table */
	t := new(table)
	t.size = 1
	copysha1(&t.items[0].sha1[0], sha1)
	t.items[0].offset = off
	t.items[0].child = *toff
	t.items[1].child = rc

	ntoff := bt.allocChunk(tableStructSize)
	bt.flush(t, ntoff)

	*toff = ntoff

	// make sure data is written before a reference is added to it
	_ = bt.fd.Sync()
	return ret
}

func (bt *Btree) lookup(toff int64, sha1 *byte) int64 {
	if toff == 0 {
		return 0
	}
	table := bt.get(toff)

	left, right := 0, table.size
	for left < right {
		mid := (right-left)>>1 + left
		switch cmp := cmp(sha1, &table.items[mid].sha1[0]); {
		case cmp == 0:
			// found
			ret := table.items[mid].offset
			bt.put(table, toff)
			return ret
		case cmp < 0:
			right = mid
		default:
			left = mid + 1
		}
	}

	i := left
	child := table.items[i].child
	bt.put(table, toff)
	return bt.lookup(child, sha1)
}

// Insert a new item with key 'sha1' with the contents in 'data' to the
// database file.
func (bt *Btree) Insert(csha1 *byte, data []byte) {
	/* SHA-1 must be in writable memory */
	var sha1 [sha1Size]byte
	copysha1(&sha1[0], csha1)

	bt.insertTopLevel(&bt.top, &sha1[0], data, len(data))
	freeQueued(bt)
	bt.flushSuper()
}

// Get look up item with the given key 'sha1' in the database file. Length of the
// item is stored in 'len'. Returns a pointer to the contents of the item.
// The returned pointer should be released with free() after use.
func (bt *Btree) Get(sha1 *byte) []byte {
	off := bt.lookup(bt.top, sha1)
	if off == 0 {
		return nil
	}

	bt.fd.Seek(off, io.SeekStart)
	var length int32
	err := binary.Read(bt.fd, binary.LittleEndian, &length)
	if err != nil {
		return nil
	}
	data := make([]byte, length)
	n, err := io.ReadFull(bt.fd, data)
	if err != nil {
		return nil
	}
	return data[:n]
}

// Delete remove item with the given key 'sha1' from the database file.
func (bt *Btree) Delete(sha1 *byte) error {
	return errors.New("impl me")
}
