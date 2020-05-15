package sync_cache

import (
	"fmt"
	"github.com/go-redis/redis"
	uuid_gen "github.com/google/uuid"
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

func (c *SyncCacheClient) Get(cacheGroupName, key string) interface{} {

	// Смотрим, есть ли объект с заданным ID-объекта в Redis
	uuid, err := c.redis.Get(cacheGroupName + "_" + key).Result()

	// Если объекта нет в Redis
	if err == redis.Nil {
		//TODO: remove debug
		fmt.Printf("key %s does not exist\n", cacheGroupName+"_"+key)

		// При наличии объекта в кэше, удалить его из кэша.
		k := c.cacheGroupManager.cacheGroups[cacheGroupName].get(key)
		if k.object != nil {
			c.cacheGroupManager.cacheGroups[cacheGroupName].delete(key)
		}

		// Взять объект из БД и добавить в кэш и в Redis.
		// Для устранения ситуации, когда обновление/удаление записи окажется быстрее,
		// чем добавление в кэш (комментарий andreyverbin) в кэш добавляем объект с нулевым UUID.
		// Тогда при первом же обращении к кэшу будет выявлена разница в UUID с Redis, а данные из БД будут
		// снова запрошены.
		setCacheFunc := func(i interface{}) {
			c.cacheGroupManager.cacheGroups[cacheGroupName].set(key, i, "")
		}

		newUuid := uuid_gen.New().String()
		c.redis.Set(cacheGroupName+"_"+key, newUuid, 0)
		c.cacheGroupManager.cacheGroups[cacheGroupName].getterFunc(setCacheFunc)

	} else if err != nil {
		panic(err)
	}

	// Если объект есть в Redis

	c.cacheGroupManager.RLock()
	if c.cacheGroupManager.cacheGroups[cacheGroupName] == nil {
		panic(fmt.Sprintf("group %s does not exist", cacheGroupName))
	}
	k := c.cacheGroupManager.cacheGroups[cacheGroupName].get(key)
	c.cacheGroupManager.RUnlock()

	setCacheFunc := func(i interface{}) {
		c.cacheGroupManager.cacheGroups[cacheGroupName].set(key, i, uuid)
	}

	// Если объекта нет в in-memory кэше, тогда берём его из БД и добавляем в in-memory кэш с UUID из Redis
	//  и обновляем TTL ключа в Redis.
	if k.object == nil {
		c.cacheGroupManager.cacheGroups[cacheGroupName].getterFunc(setCacheFunc)
	}

	// Если объект есть в in-memory кэше, тогда берём его из кэша, проверяем, совпадает ли UUID в кэше и в Redis
	//  и если да, то обновляем TTL в кэше и в Redis. Если UUID не совпадает, то удаляем объект из in-memory кэша,
	//  берём из БД, добавляем в in-memory кэш с UUID из Redis.
	if k.uuid != uuid {
		c.cacheGroupManager.cacheGroups[cacheGroupName].delete(key)
		c.cacheGroupManager.cacheGroups[cacheGroupName].getterFunc(setCacheFunc)
	}

	return c.cacheGroupManager.cacheGroups[cacheGroupName].get(key).object
}

func (c *SyncCacheClient) Delete(cacheGroupName, key string) {
	c.cacheGroupManager.RLock()
	if c.cacheGroupManager.cacheGroups[cacheGroupName] == nil {
		panic(fmt.Sprintf("group %s does not exist", cacheGroupName))
	}
	c.cacheGroupManager.cacheGroups[cacheGroupName].delete(key)
	c.cacheGroupManager.RUnlock()
}
