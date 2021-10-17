// Package btree provide a disk-based btree
package btree

import (
	"io"
	"math/rand"
	"os"
	"unsafe"

	"github.com/pkg/errors"
)

const (
	hashSize        = 16 // md5 hash
	tableSize       = (1024 - 1) / int(unsafe.Sizeof(item{}))
	cacheSlots      = 11 // prime
	superSize       = int(unsafe.Sizeof(super{}))
	tableStructSize = int(unsafe.Sizeof(table{}))
)

type item struct {
	hash   [hashSize]byte
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

// DB ...
type DB struct {
	fd      *os.File
	top     int64
	freeTop int64
	alloc   int64
	cache   [cacheSlots]cache

	inAllocator  bool
	deleteLarger bool
	fqueue       [freeQueueLen]chunk
	fqueueLen    int
}

func (d *DB) get(offset int64) *table {
	assert(offset != 0)

	// take from cache
	slot := &d.cache[offset%cacheSlots]
	if slot.offset == offset {
		return slot.table
	}

	table := new(table)

	d.fd.Seek(offset, io.SeekStart)
	err := readTable(d.fd, table)
	if err != nil {
		panic(errors.Wrap(err, "btree I/O error"))
	}
	return table
}

func (d *DB) put(t *table, offset int64) {
	assert(offset != 0)

	// overwrite cache
	slot := &d.cache[offset%cacheSlots]
	slot.table = t
	slot.offset = offset
}

func (d *DB) flush(t *table, offset int64) {
	assert(offset != 0)

	d.fd.Seek(offset, io.SeekStart)
	err := writeTable(d.fd, t)
	if err != nil {
		panic(errors.Wrap(err, "btree I/O error"))
	}
	d.put(t, offset)
}

func (d *DB) flushSuper() {
	d.fd.Seek(0, io.SeekStart)
	super := super{
		top:     d.top,
		freeTop: d.freeTop,
		alloc:   d.alloc,
	}
	err := writeSuper(d.fd, &super)
	if err != nil {
		panic(errors.Wrap(err, "btree I/O error"))
	}
}

// Open opens an existed btree file
func Open(name string) (*DB, error) {
	btree := new(DB)
	fd, err := os.OpenFile(name, os.O_RDWR, 0o644)
	if err != nil {
		return nil, errors.Wrap(err, "btree open file failed")
	}
	btree.fd = fd

	super := super{}
	err = readSuper(fd, &super)
	btree.top = super.top
	btree.freeTop = super.freeTop
	btree.alloc = super.alloc
	return btree, errors.Wrap(err, "btree read meta info failed")
}

// Create creates a database
func Create(name string) (*DB, error) {
	btree := new(DB)
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
func (d *DB) Close() error {
	_ = d.fd.Sync()
	err := d.fd.Close()
	for i := 0; i < cacheSlots; i++ {
		d.cache[i] = cache{}
	}
	return errors.Wrap(err, "btree close failed")
}

func collapse(bt *DB, offset int64) int64 {
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

// split a table. The pivot item is stored to 'hash' and 'offset'.
// Returns offset to the new table.
func (d *DB) split(t *table, hash *byte, offset *int64) int64 {
	copyhash(hash, &t.items[tableSize/2].hash[0])
	*offset = t.items[tableSize/2].offset

	ntable := new(table)
	ntable.size = t.size - tableSize/2 - 1

	t.size = tableSize / 2

	copy(ntable.items[:ntable.size+1], t.items[tableSize/2+1:])

	noff := d.allocChunk(tableStructSize)
	d.flush(ntable, noff)

	// make sure data is written before a reference is added to it
	_ = d.fd.Sync()
	return noff
}

// takeSmallest find and remove the smallest item from the given table. The key of the item
// is stored to 'hash'. Returns offset to the item
func (d *DB) takeSmallest(toff int64, sha1 *byte) int64 {
	table := d.get(toff)
	assert(table.size > 0)

	var off int64
	child := table.items[0].child
	if child == 0 {
		off = d.remove(table, 0, sha1)
	} else {
		/* recursion */
		off = d.takeSmallest(child, sha1)
		table.items[0].child = collapse(d, child)
	}
	d.flush(table, toff)

	// make sure data is written before a reference is added to it
	_ = d.fd.Sync()
	return off
}

// takeLargest find and remove the largest item from the given table. The key of the item
// is stored to 'hash'. Returns offset to the item
func (d *DB) takeLargest(toff int64, hash *byte) int64 {
	table := d.get(toff)
	assert(table.size > 0)

	var off int64
	child := table.items[table.size].child
	if child == 0 {
		off = d.remove(table, table.size-1, hash)
	} else {
		/* recursion */
		off = d.takeLargest(child, hash)
		table.items[table.size].child = collapse(d, child)
	}
	d.flush(table, toff)

	// make sure data is written before a reference is added to it
	_ = d.fd.Sync()
	return off
}

// remove an item in position 'i' from the given table. The key of the
// removed item is stored to 'hash'. Returns offset to the item.
func (d *DB) remove(t *table, i int, hash *byte) int64 {
	assert(i < t.size)

	if hash != nil {
		copyhash(hash, &t.items[i].hash[0])
	}

	offset := t.items[i].offset
	lc := t.items[i].child
	rc := t.items[i+1].child

	if lc != 0 && rc != 0 {
		/* replace the removed item by taking an item from one of the
		   child tables */
		var noff int64
		if rand.Int()&1 != 0 {
			noff = d.takeLargest(lc, &t.items[i].hash[0])
			t.items[i].child = collapse(d, lc)
		} else {
			noff = d.takeSmallest(rc, &t.items[i].hash[0])
			t.items[i+1].child = collapse(d, rc)
		}
		t.items[i].child = noff
	} else {
		// memmove(&table->items[i], &table->items[i + 1],
		//	(table->size - i) * sizeof(struct btree_item));
		copy(t.items[i:], t.items[i+1:])
		t.size--

		if lc != 0 {
			t.items[i].child = lc
		} else {
			t.items[i].child = rc
		}
	}
	return offset
}

func (d *DB) insert(toff int64, hash *byte, data []byte, size int) int64 {
	table := d.get(toff)
	assert(table.size < tableSize-1)

	left, right := 0, table.size
	for left < right {
		mid := (right-left)>>1 + left
		switch cmp := cmp(hash, &table.items[mid].hash[0]); {
		case cmp == 0:
			// already in the table
			ret := table.items[mid].offset
			d.put(table, toff)
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
		ret = d.insert(lc, hash, data, size)

		/* check if we need to split */
		child := d.get(lc)
		if child.size < tableSize-1 {
			/* nothing to do */
			d.put(table, toff)
			d.put(child, lc)
			return ret
		}
		/* overwrites SHA-1 */
		rc = d.split(child, hash, &off)
		/* flush just in case changes happened */
		d.flush(child, lc)

		// make sure data is written before a reference is added to it
		_ = d.fd.Sync()
	} else {
		off = d.insertData(data, size)
		ret = off
	}

	table.size++
	// memmove(&table->items[i + 1], &table->items[i],
	//  (table->size - i) * sizeof(struct btree_item));
	copy(table.items[i+1:], table.items[i:])
	copyhash(&table.items[i].hash[0], hash)
	table.items[i].offset = off
	table.items[i].child = lc
	table.items[i+1].child = rc

	d.flush(table, toff)
	return ret
}

func (d *DB) insertData(data []byte, size int) int64 {
	if data == nil {
		return int64(size)
	}
	assert(len(data) == size)

	offset := d.allocChunk(4 + len(data))

	d.fd.Seek(offset, io.SeekStart)
	err := write32(d.fd, int32(len(data)))
	if err != nil {
		panic(errors.Wrap(err, "btree I/O error"))
	}
	_, err = d.fd.Write(data)
	if err != nil {
		panic(errors.Wrap(err, "btree I/O error"))
	}

	// make sure data is written before a reference is added to it
	_ = d.fd.Sync()
	return offset
}

// delete remove an item with key 'hash' from the given table. The offset to the
// removed item is returned.
// Please note that 'hash' is overwritten when called inside the allocator.
func (d *DB) delete(offset int64, hash *byte) int64 {
	if offset == 0 {
		return 0
	}
	table := d.get(offset)

	left, right := 0, table.size
	for left < right {
		i := (right-left)>>1 + left
		switch cmp := cmp(hash, &table.items[i].hash[0]); {
		case cmp == 0:
			// found
			ret := d.remove(table, i, hash)
			d.flush(table, offset)
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
	ret := d.delete(child, hash)
	if ret != 0 {
		table.items[i].child = collapse(d, child)
	}

	if ret == 0 && d.deleteLarger && i < table.size {
		ret = d.remove(table, i, hash)
	}
	if ret != 0 {
		/* flush just in case changes happened */
		d.flush(table, offset)
	} else {
		d.put(table, offset)
	}
	return ret
}

func (d *DB) insertTopLevel(toff *int64, hash *byte, data []byte, size int) int64 { // nolint:unparam
	var off, ret, rc int64
	if *toff != 0 {
		ret = d.insert(*toff, hash, data, size)

		/* check if we need to split */
		table := d.get(*toff)
		if table.size < tableSize-1 {
			/* nothing to do */
			d.put(table, *toff)
			return ret
		}
		rc = d.split(table, hash, &off)
		d.flush(table, *toff)
	} else {
		off = d.insertData(data, size)
		ret = off
	}

	/* create new top level table */
	t := new(table)
	t.size = 1
	copyhash(&t.items[0].hash[0], hash)
	t.items[0].offset = off
	t.items[0].child = *toff
	t.items[1].child = rc

	ntoff := d.allocChunk(tableStructSize)
	d.flush(t, ntoff)

	*toff = ntoff

	// make sure data is written before a reference is added to it
	_ = d.fd.Sync()
	return ret
}

func (d *DB) lookup(toff int64, hash *byte) int64 {
	if toff == 0 {
		return 0
	}
	table := d.get(toff)

	left, right := 0, table.size
	for left < right {
		mid := (right-left)>>1 + left
		switch cmp := cmp(hash, &table.items[mid].hash[0]); {
		case cmp == 0:
			// found
			ret := table.items[mid].offset
			d.put(table, toff)
			return ret
		case cmp < 0:
			right = mid
		default:
			left = mid + 1
		}
	}

	i := left
	child := table.items[i].child
	d.put(table, toff)
	return d.lookup(child, hash)
}

// Insert a new item with key 'hash' with the contents in 'data' to the
// database file.
func (d *DB) Insert(chash *byte, data []byte) {
	/* SHA-1 must be in writable memory */
	var hash [hashSize]byte
	copyhash(&hash[0], chash)

	_ = d.insertTopLevel(&d.top, &hash[0], data, len(data))
	freeQueued(d)
	d.flushSuper()
}

// Get look up item with the given key 'hash' in the database file. Length of the
// item is stored in 'len'. Returns a pointer to the contents of the item.
// The returned pointer should be released with free() after use.
func (d *DB) Get(hash *byte) []byte {
	off := d.lookup(d.top, hash)
	if off == 0 {
		return nil
	}

	d.fd.Seek(off, io.SeekStart)
	length, err := read32(d.fd)
	if err != nil {
		return nil
	}
	data := make([]byte, length)
	n, err := io.ReadFull(d.fd, data)
	if err != nil {
		return nil
	}
	return data[:n]
}

// Delete remove item with the given key 'hash' from the database file.
func (d *DB) Delete(hash *byte) error {
	var h [hashSize]byte
	copyhash(&h[0], hash)

	off := d.delete(d.top, &h[0])
	if off == 0 {
		return nil // not found key
	}

	d.top = collapse(d, d.top)
	freeQueued(d)
	d.flushSuper()

	d.fd.Seek(off, io.SeekStart)
	length, err := read32(d.fd) // len: 0
	if err != nil {
		return errors.Wrap(err, "btree I/O error")
	}

	d.freeChunk(off, int(length+4))
	freeQueued(d)
	d.flushSuper()
	return nil
}
