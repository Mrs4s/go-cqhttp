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

// todo(wdvxdr): move this to btree?
var (
	fqueue    [freeQueueLen]chunk
	fqueueLen = 0
)

func freeQueued(bt *Btree) {
	for i := 0; i < fqueueLen; i++ {
		chunk := &fqueue[i]
		bt.freeChunk(chunk.offset, chunk.len)
	}
	fqueueLen = 0
}

func (bt *Btree) allocChunk(size int) int64 {
	assert(size > 0)

	size = power2(size)

	var offset int64
	if bt.inAllocator {
		const i32s = unsafe.Sizeof(int32(0))

		/* create fake size SHA-1 */
		var sha1 [sha1Size]byte
		p := unsafe.Pointer(&sha1[0])
		*(*int32)(p) = -1                              // *(uint32_t *) sha1 = -1;
		*(*uint32)(unsafe.Add(p, i32s)) = uint32(size) // ((__be32 *) sha1)[1] = to_be32(size);

		/* find free chunk with the larger or the same size/SHA-1 */
		bt.inAllocator = true
		bt.deleteLarger = true
		offset = bt.delete(bt.freeTop, &sha1[0])
		bt.deleteLarger = false
		if offset != 0 {
			assert(*(*int32)(p) == -1)                   // assert(*(uint32_t *) sha1 == (uint32_t) -1)
			flen := int(*(*uint32)(unsafe.Add(p, i32s))) // size_t free_len = from_be32(((__be32 *) sha1)[1])
			assert(power2(flen) == flen)
			assert(flen >= size)

			/* delete buddy information */
			resetsha1(&sha1[0])
			*(*int64)(p) = offset
			buddyLen := bt.delete(bt.freeTop, &sha1[0])
			assert(buddyLen == int64(size))

			bt.freeTop = collapse(bt, bt.freeTop)

			bt.inAllocator = false

			/* free extra space at the end of the chunk */
			for flen > size {
				flen >>= 1
				bt.freeChunk(offset+int64(flen), flen)
			}
		} else {
			bt.inAllocator = false
		}
	}
	if offset == 0 {
		/* not found, allocate from the end of the file */
		offset = bt.alloc
		/* TODO: this wastes memory.. */
		if offset&int64(size-1) != 0 {
			offset += int64(size) - (offset & (int64(size) - 1))
		}
		bt.alloc = offset + int64(size)
	}
	bt.flushSuper()

	// make sure the allocation tree is up-to-date before using the chunk
	_ = bt.fd.Sync()
	return offset
}

/* Mark a chunk as unused in the database file */
func (bt *Btree) freeChunk(offset int64, size int) {
	assert(size > 0)
	assert(offset != 0)
	size = power2(size)
	assert(offset&int64(size-1) == 0)

	if bt.inAllocator {
		chunk := &fqueue[fqueueLen]
		fqueueLen++
		chunk.offset = offset
		chunk.len = size
		return
	}

	/* create fake offset SHA-1 for buddy allocation */
	var sha1 [sha1Size]byte
	p := unsafe.Pointer(&sha1[0])
	bt.inAllocator = true

	const i32s = unsafe.Sizeof(int32(0))

	/* add buddy information */
	resetsha1(&sha1[0])
	*(*int32)(p) = -1                                 // *(uint32_t *) sha1 = -1;
	*(*uint32)(unsafe.Add(p, i32s)) = uint32(size)    // ((__be32 *) sha1)[1] = to_be32(size);
	*(*uint32)(unsafe.Add(p, i32s*2)) = rand.Uint32() /* to make SHA-1 unique */
	*(*uint32)(unsafe.Add(p, i32s*3)) = rand.Uint32()

	// insert_toplevel(btree, &btree->free_top, sha1, NULL, offset);
	_ = bt.insertTopLevel(&bt.freeTop, &sha1[0], nil, int(offset))
	bt.inAllocator = false

	bt.flushSuper()

	// make sure the allocation tree is up-to-date before removing
	// references to the chunk
	_ = bt.fd.Sync()
}
