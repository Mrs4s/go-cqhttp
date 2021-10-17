package btree

import (
	"math/rand"
	"unsafe"
)

type chunk struct {
	offset int64
	len    int
}

const freeQueueLen = 64

func freeQueued(bt *DB) {
	for i := 0; i < bt.fqueueLen; i++ {
		chunk := &bt.fqueue[i]
		bt.freeChunk(chunk.offset, chunk.len)
	}
	bt.fqueueLen = 0
}

func (d *DB) allocChunk(size int) int64 {
	assert(size > 0)

	size = power2(size)

	var offset int64
	if d.inAllocator {
		const i32s = unsafe.Sizeof(int32(0))

		/* create fake size SHA-1 */
		var sha1 [hashSize]byte
		p := unsafe.Pointer(&sha1[0])
		*(*int32)(p) = -1                              // *(uint32_t *) hash = -1;
		*(*uint32)(unsafe.Add(p, i32s)) = uint32(size) // ((__be32 *) hash)[1] = to_be32(size);

		/* find free chunk with the larger or the same size/SHA-1 */
		d.inAllocator = true
		d.deleteLarger = true
		offset = d.delete(d.freeTop, &sha1[0])
		d.deleteLarger = false
		if offset != 0 {
			assert(*(*int32)(p) == -1)                   // assert(*(uint32_t *) hash == (uint32_t) -1)
			flen := int(*(*uint32)(unsafe.Add(p, i32s))) // size_t free_len = from_be32(((__be32 *) hash)[1])
			assert(power2(flen) == flen)
			assert(flen >= size)

			/* delete buddy information */
			resethash(&sha1[0])
			*(*int64)(p) = offset
			buddyLen := d.delete(d.freeTop, &sha1[0])
			assert(buddyLen == int64(size))

			d.freeTop = collapse(d, d.freeTop)

			d.inAllocator = false

			/* free extra space at the end of the chunk */
			for flen > size {
				flen >>= 1
				d.freeChunk(offset+int64(flen), flen)
			}
		} else {
			d.inAllocator = false
		}
	}
	if offset == 0 {
		/* not found, allocate from the end of the file */
		offset = d.alloc
		/* TODO: this wastes memory.. */
		if offset&int64(size-1) != 0 {
			offset += int64(size) - (offset & (int64(size) - 1))
		}
		d.alloc = offset + int64(size)
	}
	d.flushSuper()

	// make sure the allocation tree is up-to-date before using the chunk
	_ = d.fd.Sync()
	return offset
}

/* Mark a chunk as unused in the database file */
func (d *DB) freeChunk(offset int64, size int) {
	assert(size > 0)
	assert(offset != 0)
	size = power2(size)
	assert(offset&int64(size-1) == 0)

	if d.inAllocator {
		chunk := &d.fqueue[d.fqueueLen]
		d.fqueueLen++
		chunk.offset = offset
		chunk.len = size
		return
	}

	/* create fake offset SHA-1 for buddy allocation */
	var sha1 [hashSize]byte
	p := unsafe.Pointer(&sha1[0])
	d.inAllocator = true

	const i32s = unsafe.Sizeof(int32(0))

	/* add buddy information */
	resethash(&sha1[0])
	*(*int32)(p) = -1                                 // *(uint32_t *) hash = -1;
	*(*uint32)(unsafe.Add(p, i32s)) = uint32(size)    // ((__be32 *) hash)[1] = to_be32(size);
	*(*uint32)(unsafe.Add(p, i32s*2)) = rand.Uint32() /* to make SHA-1 unique */
	*(*uint32)(unsafe.Add(p, i32s*3)) = rand.Uint32()

	// insert_toplevel(btree, &btree->free_top, hash, NULL, offset);
	_ = d.insertTopLevel(&d.freeTop, &sha1[0], nil, int(offset))
	d.inAllocator = false

	d.flushSuper()

	// make sure the allocation tree is up-to-date before removing
	// references to the chunk
	_ = d.fd.Sync()
}
