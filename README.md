# sync-cache
Multi-instance cache

This cache uses redis for sync

Inspired by https://habr.com/ru/post/482704/


## Usage

### Example

```go
import (
 "github.com/inhuman/sync-cache"
)
```

Then run `docker-compose up` for create local docker redis

```go
func main() {
	cacheClient := NewSyncCacheClient(SyncCacheOpts{
    		Address: "localhost:6379",
    		Db:      10,
    	})
    
    	getterFunc := func(key string, setCacheFunc SetCacheFunc) error {
    		obj, err := fetchObjectFromSource(key)
            if err != nil {
                return err
            }
       
    		setCacheFunc(obj)
    		return nil
    	}
    
    	cacheClient.AddCacheGroup("cacheGroupName", getterFunc)

        i, err := cacheClient.Get(groupName, "testKey")
        if err != nil {
            panic(err)
        }   
        
        fmt.Printf("object: %+v\n", i)

        n, err := cacheClient.Delete(groupName, "testKey")
        if err != nil {
            panic(err)
        }   
    
    fmt.Printf("n: %d\n", n)
}
```