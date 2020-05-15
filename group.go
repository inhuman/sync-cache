package sync_cache

import "sync"

const GroupCacheCapacity = 100

type GetterFunc func(setCacheFunc SetCacheFunc)
type SetCacheFunc func(i interface{})

type CacheGroup struct {
	sync.RWMutex
	cache      map[string]CacheEntity
	getterFunc GetterFunc
}

type CacheEntity struct {
	uuid   string
	object interface{}
}

func NewCacheGroup(getterFunc GetterFunc) *CacheGroup {
	return &CacheGroup{
		cache:      make(map[string]CacheEntity, GroupCacheCapacity),
		getterFunc: getterFunc,
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
