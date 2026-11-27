package service

import (
	"container/list"
	"sync"
	"time"
)

// TileKey names one map tile.
type TileKey struct {
	Zoom, X, Y int
}

type cachedTile struct {
	key    TileKey
	data   []byte
	stored time.Time
}

// TileCache holds the PNG bytes of the map tiles in memory, so a restart of the hub
// empties it. It drops the least recently used tile when the bytes pass the limit, and
// drops a tile when its age passes the time to live.
type TileCache struct {
	mu      sync.Mutex
	limit   int
	ttl     time.Duration
	now     func() time.Time
	size    int
	order   *list.List
	entries map[TileKey]*list.Element
}

// NewTileCache creates a TileCache that holds at most limit bytes, each tile for ttl.
func NewTileCache(limit int, ttl time.Duration) *TileCache {
	return &TileCache{
		limit:   limit,
		ttl:     ttl,
		now:     time.Now,
		order:   list.New(),
		entries: map[TileKey]*list.Element{},
	}
}

// Get answers the bytes of the tile, and false when the cache does not hold it.
func (c *TileCache) Get(key TileKey) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, found := c.entries[key]
	if !found {
		return nil, false
	}

	tile, _ := element.Value.(*cachedTile)
	if c.now().Sub(tile.stored) > c.ttl {
		c.remove(element)

		return nil, false
	}

	c.order.MoveToFront(element)

	return tile.data, true
}

// Put stores the bytes of the tile. A tile larger than the limit stays out.
func (c *TileCache) Put(key TileKey, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, found := c.entries[key]; found {
		c.remove(element)
	}

	if len(data) > c.limit {
		return
	}

	c.entries[key] = c.order.PushFront(&cachedTile{key: key, data: data, stored: c.now()})
	c.size += len(data)

	for c.size > c.limit {
		c.remove(c.order.Back())
	}
}

// Size answers the sum of the bytes that the cache holds.
func (c *TileCache) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.size
}

func (c *TileCache) remove(element *list.Element) {
	tile, _ := c.order.Remove(element).(*cachedTile)
	delete(c.entries, tile.key)
	c.size -= len(tile.data)
}
