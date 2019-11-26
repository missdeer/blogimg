// main logic:
// determine images from post
// first, check redis records, if there is a image link for this post, return directly
// otherwise, check post content, try to extract images from post, get the first large image ( w > 480 && h > 320 )
// then, if there is no image that is good enough, select one from Bing Wallpaper
// at last, save the post link & image link to redis
package main

import (
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	defaultImage = `https://cdn.jsdelivr.net/gh/missdeer/blog@gh-pages/assets/images/logo.png`
)

var (
	cache *RedisCache
)

// handleImageRequestForPost returns an image for the specified blog post
// request examples:
// https://blogimage.minidump.info/2018-07-10-clang-on-windows-for-qt.md
// https://blogimage.minidump.info/2006-09-10-%e5%a4%a7%e9%9b%84%e7%9a%84%e7%bb%93%e5%a9%9a%e5%89%8d%e5%a4%9c.html
func handleImageRequestForPost(c *gin.Context) {
	post := c.Param("post")

	result, err := readFromRedis(post)
	if err == nil {
		c.Redirect(http.StatusFound, result)
		cache.Put(post, result)
		return
	}
	log.Println(err)

	result, err = extractFromPostContent(post)
	if err == nil {
		c.Redirect(http.StatusFound, result)
		cache.Put(post, result)
		return
	}
	log.Println(err)

	result, err = pickFromBingWallpaper(post, true)
	if err == nil {
		c.Redirect(http.StatusFound, result)
		cache.Put(post, result)
		return
	}
	log.Println(err)
	c.Redirect(http.StatusFound, defaultImage)
}

func handleDeleteCachedPostImage(c *gin.Context) {
	post := c.Param("post")
	if cache.IsExist(post) {
		if err := cache.Delete(post); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"result":  "error",
				"message": err.Error(),
			})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"result": "OK",
	})
}

func handleClearAllCachedPostImage(c *gin.Context) {
	if err := cache.ClearAll(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"result":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"result": "OK",
	})
}

func handleUpdatePostImage(c *gin.Context) {
	post := c.Query("post")
	img := c.Query("img")
	_, err := url.Parse(img)
	if post == "" || err != nil {
		errMsg := "invalid post name or image URL"
		if err != nil {
			errMsg = err.Error()
		}
		c.JSON(http.StatusOK, gin.H{
			"result":  "error",
			"message": errMsg,
		})
		return
	}

	cache.Put(post, img)

	c.JSON(http.StatusOK, gin.H{
		"result": "OK",
	})
}

func main() {
	cache = RedisInit()
	rand.Seed(time.Now().Unix())
	bindAddr := os.Getenv("BIND_ADDR")
	if bindAddr == "" {
		bindAddr = "127.0.0.1:8585"
	}
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "https://minidump.info/blog/")
	})
	r.GET("/:post", handleImageRequestForPost)

	user := os.Getenv("USER")
	if user == "" {
		log.Fatal("environment variable USER is required")
	}
	passwd := os.Getenv("PASSWD")
	if passwd == "" {
		log.Fatal("environment variable PASSWD is required")
	}
	authorized := r.Group("/admin", gin.BasicAuth(gin.Accounts{
		user: passwd,
	}))
	authorized.DELETE("/:post", handleDeleteCachedPostImage)
	authorized.POST("/clearall", handleClearAllCachedPostImage)
	authorized.POST("/update", handleUpdatePostImage)
	log.Fatal(r.Run(bindAddr))
}
