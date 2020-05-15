package sync_cache

import (
	"fmt"
	"github.com/go-redis/redis"
	uuidgen "github.com/google/uuid"
	"sync"
)

const SyncCacheGroupCapacity = 10

type SyncCacheOpts struct {
	Address  string
	Password string
	Db       int
}

type SyncCacheClient struct {
	redis             *redis.Client
	cacheGroupManager *CacheGroupsManager
}

type CacheGroupsManager struct {
	cacheGroups map[string]*CacheGroup
	sync.RWMutex
}

func NewSyncCacheClient(opts SyncCacheOpts) *SyncCacheClient {

	c := &SyncCacheClient{
		cacheGroupManager: &CacheGroupsManager{
			cacheGroups: make(map[string]*CacheGroup, SyncCacheGroupCapacity),
		},
	}

	c.redis = redis.NewClient(&redis.Options{
		Addr:     opts.Address,
		Password: opts.Password,
		DB:       opts.Db,
	})

	return c
}

func (c *SyncCacheClient) AddCacheGroup(cacheGroupName string, getterFunc GetterFunc) {
	c.cacheGroupManager.Lock()
	c.cacheGroupManager.cacheGroups[cacheGroupName] = NewCacheGroup(getterFunc)
	c.cacheGroupManager.Unlock()
}

func (c *SyncCacheClient) RemoveCacheGroup(cacheGroupName string) {
	c.cacheGroupManager.Lock()
	delete(c.cacheGroupManager.cacheGroups, cacheGroupName)
	c.cacheGroupManager.Unlock()
}

func (c *SyncCacheClient) UpdateRedisKey(cacheGroupName, key string) {
	newUuid := uuidgen.New().String()
	c.redis.Set(cacheGroupName+"_"+key, newUuid, 0)
}

func (c *SyncCacheClient) Get(cacheGroupName, key string) (interface{}, error) {

	// Try to find an record with a given ID object in Redis
	uuid, err := c.redis.Get(cacheGroupName + "_" + key).Result()

	// Record not found in Redis
	if err == redis.Nil {

		// If object exists in cache - remove it
		k := c.cacheGroupManager.cacheGroups[cacheGroupName].get(key)
		if k.object != nil {
			c.cacheGroupManager.cacheGroups[cacheGroupName].delete(key)
		}

		// Fetch object from db and add to cache
		// To avoid the situation when updating / deleting a record is faster,
		// than adding to the cache (andreyverbin comment) we add an object with a zero UUID to the cache.
		// Then the first time you access the cache, the difference in UUID with Redis will be revealed,
		// and the data from the database will be requested again.
		setCacheFunc := func(i interface{}) {
			c.cacheGroupManager.cacheGroups[cacheGroupName].set(key, i, "")
		}

		// and to Redis
		newUuid := uuidgen.New().String()
		c.redis.Set(cacheGroupName+"_"+key, newUuid, 0)

		if err := c.cacheGroupManager.cacheGroups[cacheGroupName].getterFunc(key, setCacheFunc); err != nil {
			return nil, err
		}

	} else if err != nil {
		return nil, err
	}

	// If object exists in Redis
	c.cacheGroupManager.RLock()
	if c.cacheGroupManager.cacheGroups[cacheGroupName] == nil {
		panic(fmt.Sprintf("group %s does not exist", cacheGroupName))
	}
	k := c.cacheGroupManager.cacheGroups[cacheGroupName].get(key)
	c.cacheGroupManager.RUnlock()

	setCacheFunc := func(i interface{}) {
		c.cacheGroupManager.cacheGroups[cacheGroupName].set(key, i, uuid)
	}

	// If object not exists in memory cache than fetch it from db and add to memory cache with Redis uuid
	if k.object == nil {
		if err := c.cacheGroupManager.cacheGroups[cacheGroupName].getterFunc(key, setCacheFunc); err != nil {
			return nil, err
		}
	}

	// If object exists in memory cache, than take it from cache, compare uuid in cache with Redis uuid
	// if not, delete object from memory cache, and add object from db to cache with Redis uuid
	if k.uuid != uuid {
		c.cacheGroupManager.cacheGroups[cacheGroupName].delete(key)
		if err := c.cacheGroupManager.cacheGroups[cacheGroupName].getterFunc(key, setCacheFunc); err != nil {
			return nil, err
		}
	}

	return c.cacheGroupManager.cacheGroups[cacheGroupName].get(key).object, nil
}

func (c *SyncCacheClient) Delete(cacheGroupName, key string) (int64, error) {

	c.cacheGroupManager.RLock()
	if c.cacheGroupManager.cacheGroups[cacheGroupName] == nil {
		panic(fmt.Sprintf("group %s does not exist", cacheGroupName))
	}
	c.cacheGroupManager.cacheGroups[cacheGroupName].delete(key)
	c.cacheGroupManager.RUnlock()

	return c.redis.Del(cacheGroupName + "_" + key).Result()
}
