package main

import (
	"math/rand"
	"testing"
	"time"
)

func TestPickFromBingWallpaper(t *testing.T) {
	cache = RedisInit()
	rand.Seed(time.Now().Unix())
	result, err := pickFromBingWallpaper(`2019-09-17-a-new-markdown-editor.md`)
	if err != nil {
		t.Error(err)
	}
	t.Log(result)
}
