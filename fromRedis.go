package main

import (
	"errors"

	"github.com/garyburd/redigo/redis"
)

func readFromRedis(post string) (string, error) {
	// read from redis
	if cache.IsExist(post) {
		if imageLink, err := redis.String(cache.Get(post)); err == nil {
			return imageLink, nil
		} else {
			return "", err
		}
	}
	return "", errors.New("not found in redis")
}
