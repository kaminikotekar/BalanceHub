package Cache

import (
	"fmt"
	"github.com/kaminikotekar/BalanceHub/pkg/Redis"
	"net/http"
	"time"
)

func generateCacheKey(req *http.Request) string {

	url := req.URL.Path
	method := req.Method
	body := req.Body
	if method == "GET" {
		return fmt.Sprintf("request:%s:%s", url, body)
	}
	return ""
}

func GetFromCache(req *http.Request) ([]byte, error) {
	key := generateCacheKey(req)

	response, err := Redis.GetRDClient().Get(Redis.GetContext(), key).Result()

	return []byte(response), err
}

func CacheResponse(req *http.Request, response []byte) error {
	if !Redis.IsCacheAllowed() {
		return nil
	}
	key := generateCacheKey(req)
	return cacheToDb(key, response)
}

func cacheToDb(key string, response []byte) error {
	cacheDuration := time.Duration(Redis.CacheDuration()) * time.Second
	return Redis.GetRDClient().Set(Redis.GetContext(), key, response, cacheDuration).Err()
}
