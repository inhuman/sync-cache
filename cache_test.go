package sync_cache

import (
	uuid_gen "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMain(m *testing.M) {

	m.Run()
}

//
//func TestRedis(t *testing.T) {
//
//	client := redis.NewClient(&redis.Options{
//		Addr:     "localhost:6379",
//		Password: "", // no password set
//		DB:       10,  // use default DB
//	})
//	err := client.Set("key", "value1", 0).Err()
//	if err != nil {
//		panic(err)
//	}
//
//	val, err := client.Get("key").Result()
//	if err != nil {
//		panic(err)
//	}
//	fmt.Println("key", val)
//
//	val2, err := client.Get("key2").Result()
//	if err == redis.Nil {
//		fmt.Println("key2 does not exist")
//	} else if err != nil {
//		panic(err)
//	} else {
//		fmt.Println("key2", val2)
//	}
//	// Output: key value
//	// key2 does not exist
//}

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
	cacheClient := NewSyncCacheClient(SyncCacheOpts{})

	getterFunc := func(setCacheFunc SetCacheFunc) {
		obj := newTestObj()
		setCacheFunc(obj)
	}

	cacheClient.AddCacheGroup(groupName, getterFunc)
	return cacheClient
}

func BenchmarkSyncCacheClient_Get(b *testing.B) {
	b.ReportAllocs()

	cacheClient := newClientWithGroup()
	cacheClient.redis.Set("test_test", "1", 0)

	for i := 0; i < b.N; i++ {
		cacheClient.Get(groupName, "test")
	}
}

// Если объект есть в Redis
// Если объекта нет в in-memory кэше
func TestSyncCacheClient_Get1(t *testing.T) {

	cacheClient := newClientWithGroup()
	cacheClient.redis.Set("testGroup_testKey", "redis_uuid", 0)

	i := cacheClient.Get(groupName, "testKey")

	obj, ok := i.(testStruct)
	assert.True(t, ok)
	assert.Equal(t, 1000, obj.Param2)

	cacheEntity := cacheClient.cacheGroupManager.cacheGroups[groupName].get("testKey")
	assert.Equal(t, "redis_uuid", cacheEntity.uuid)
}

// Если объекта нет в Redis
// Если объекта нет в in-memory кэше
func TestSyncCacheClient_Get2(t *testing.T) {

	cacheClient := newClientWithGroup()
	cacheClient.redis.Del("testGroup_testKey")

	i := cacheClient.Get(groupName, "testKey")

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

// Если объекта нет в Redis
// При наличии объекта в кэше
func TestSyncCacheClient_Get3(t *testing.T) {

	cacheClient := newClientWithGroup()
	objOld := newTestObj()
	objOld.Param2 = 111
	cacheClient.cacheGroupManager.cacheGroups[groupName].set("testKey", objOld, "old_redis_uuid")

	cacheClient.redis.Set("testGroup_testKey", "redis_uuid", 0)

	i := cacheClient.Get(groupName, "testKey")

	obj, ok := i.(testStruct)
	assert.True(t, ok)
	assert.Equal(t, 1000, obj.Param2)

	cacheEntity := cacheClient.cacheGroupManager.cacheGroups[groupName].get("testKey")
	assert.Equal(t, "redis_uuid", cacheEntity.uuid)
}
