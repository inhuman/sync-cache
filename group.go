package sync_cache

import "sync"

const GroupCacheCapacity = 100

type GetterFunc func(key string, setCacheFunc SetCacheFunc) error
type SetCacheFunc func(i interface{})

type CacheGroup struct {
	name string
	sync.RWMutex
	cache      map[string]CacheEntity
	getterFunc GetterFunc
}

type CacheEntity struct {
	uuid   string
	object interface{}
}

type CacheStats struct {
	Items int
}

func NewCacheGroup(name string, getterFunc GetterFunc) *CacheGroup {
	return &CacheGroup{
		name:       name,
		cache:      make(map[string]CacheEntity, GroupCacheCapacity),
		getterFunc: getterFunc,
	}
}

func (g *CacheGroup) Name() string {
	return g.name
}

func (g *CacheGroup) CacheStats() *CacheStats {
	return &CacheStats{
		Items: len(g.cache),
	}
}

func (g *CacheGroup) set(key string, i interface{}, uuid string) {
	g.Lock()
	g.cache[key] = CacheEntity{
		object: i,
		uuid:   uuid,
	}
	g.Unlock()
}

func (g *CacheGroup) get(key string) CacheEntity {
	g.RLock()
	k := g.cache[key]
	g.RUnlock()

	return k
}

func (g *CacheGroup) delete(key string) {
	g.Lock()
	delete(g.cache, key)
	g.Unlock()
}
