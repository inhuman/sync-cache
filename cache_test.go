package sync_cache

import (
	"fmt"
	"github.com/go-redis/redis"
	uuid_gen "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestMain(m *testing.M) {

	m.Run()
	os.Exit(0)
}

func TestRedis(t *testing.T) {

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       10,
	})
	err := client.Set("key", "value1", 0).Err()
	if err != nil {
		panic(err)
	}

	val, err := client.Get("key").Result()
	if err != nil {
		panic(err)
	}
	fmt.Println("key", val)

	val2, err := client.Get("key2").Result()
	if err == redis.Nil {
		fmt.Println("key2 does not exist")
	} else if err != nil {
		panic(err)
	} else {
		fmt.Println("key2", val2)
	}
	// Output: key value
	// key2 does not exist
}

type testStruct struct {
	Param1 string
	Param2 int
}

const groupName = "testGroup"

func newTestObj() testStruct {
	return testStruct{
		Param1: "testStr",
		Param2: 1000,
	}
}

func newClientWithGroup() *SyncCacheClient {
	cacheClient := NewSyncCacheClient(SyncCacheOpts{
		Address: "localhost:6379",
		Db:      10,
	})

	getterFunc := func(key string, setCacheFunc SetCacheFunc) error {
		obj := newTestObj()
		setCacheFunc(obj)
		return nil
	}

	cacheClient.AddCacheGroup(groupName, getterFunc)
	return cacheClient
}

func BenchmarkSyncCacheClient_Get(b *testing.B) {
	b.ReportAllocs()

	cacheClient := newClientWithGroup()
	cacheClient.redis.Set("testGroup_testKey", "1", 0)
	defer cacheClient.redis.FlushDB()

	for i := 0; i < b.N; i++ {
		if _, err := cacheClient.Get(groupName, "test"); err != nil {
			b.Error(err)
		}
	}
}

// Если объект есть в Redis
// Если объекта нет в in-memory кэше
func TestSyncCacheClient_Get1(t *testing.T) {

	cacheClient := newClientWithGroup()
	cacheClient.redis.Set("testGroup_testKey", "redis_uuid", 0)
	defer cacheClient.redis.FlushDB()

	i, err := cacheClient.Get(groupName, "testKey")
	assert.NoError(t, err)

	obj, ok := i.(testStruct)
	assert.True(t, ok)
	assert.Equal(t, 1000, obj.Param2)

	cacheEntity := cacheClient.cacheGroupManager.cacheGroups[groupName].get("testKey")
	assert.Equal(t, "redis_uuid", cacheEntity.uuid)
}

// Key not in Redis
// Object not in cache
func TestSyncCacheClient_Get2(t *testing.T) {

	cacheClient := newClientWithGroup()
	defer cacheClient.redis.FlushDB()

	i, err := cacheClient.Get(groupName, "testKey")
	assert.NoError(t, err)

	obj, ok := i.(testStruct)
	assert.True(t, ok)
	assert.Equal(t, 1000, obj.Param2)

	cacheEntity := cacheClient.cacheGroupManager.cacheGroups[groupName].get("testKey")
	assert.Equal(t, "", cacheEntity.uuid)

	uuidFromRedis, err := cacheClient.redis.Get("testGroup_testKey").Result()
	assert.NoError(t, err)

	_, err = uuid_gen.Parse(uuidFromRedis)
	assert.NoError(t, err)
}

// Key not in Redis
// Object in cache
func TestSyncCacheClient_Get3(t *testing.T) {

	cacheClient := newClientWithGroup()
	objOld := newTestObj()
	objOld.Param2 = 111
	cacheClient.cacheGroupManager.cacheGroups[groupName].set("testKey", objOld, "old_redis_uuid")

	cacheClient.redis.Set("testGroup_testKey", "redis_uuid", 0)
	defer cacheClient.redis.FlushDB()

	i, err := cacheClient.Get(groupName, "testKey")
	assert.NoError(t, err)

	obj, ok := i.(testStruct)
	assert.True(t, ok)
	assert.Equal(t, 1000, obj.Param2)

	cacheEntity := cacheClient.cacheGroupManager.cacheGroups[groupName].get("testKey")
	assert.Equal(t, "redis_uuid", cacheEntity.uuid)
}

// Key not in Redis
// Object in cache
// Object not in db
func TestSyncCacheClient_Get4(t *testing.T) {

	cacheClient := NewSyncCacheClient(SyncCacheOpts{})
	defer cacheClient.redis.FlushDB()

	getterFunc := func(key string, setCacheFunc SetCacheFunc) error {
		return fmt.Errorf("test error")
	}

	cacheClient.AddCacheGroup(groupName, getterFunc)

	_, err := cacheClient.Get(groupName, "testKey")
	assert.Error(t, err)
}
