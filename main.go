// main logic:
// determine images from post
// first, check redis records, if there is a image link for this post, return directly
// otherwise, check post content, try to extract images from post, get the first large image ( w > 480 && h > 320 )
// then, if there is no image that is good enough, select one from Bing Wallpaper
// at last, save the post link & image link to redis
package main

import (
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
)

const (
	defaultImage            = `https://cdn.jsdelivr.net/gh/missdeer/blog@gh-pages/assets/images/logo.png`
	postPattern             = `^(\d{4})\-(\d{2})\-(\d{2})\-([^\.]+)\.(html|md)$`
	peapixURLTemplate       = `https://peapix.com/bing?year=%s&tag=%s`
	peapixNoYearURLTemplate = `https://peapix.com/bing?tag=%s`
	peapixPattern           = `^https:\/\/img\.peapix\.com\/[0-9a-z]+_[0-9]{3}\.jpg$`
	minWidth                = 480
	minHeight               = 320
)

var (
	cache       *RedisCache
	regexPost   = regexp.MustCompile(postPattern)
	regexPeapix = regexp.MustCompile(peapixPattern)
	tags        = []string{
		"water", "outdoor", "sky", "lake", "landscape", "nature", "cloud", "mountain", "beach", "tree",
		"surrounded", "reflection", "ocean",
	}
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

func extractFromPostContent(post string) (string, error) {
	// extract from post content
	ss := regexPost.FindAllStringSubmatch(post, -1)
	if len(ss) == 0 || len(ss[0]) != 6 {
		return "", errors.New("unexpected parameter")
	}
	targetURL := fmt.Sprintf(`https://minidump.info/blog/%s/%s/%s/`, ss[0][1], ss[0][2], ss[0][4])
	resp, err := http.Get(targetURL)
	if err != nil {
		return "", errors.New("can't read post content")
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	found := false
	var result string
	doc.Find("body img").Each(func(i int, s *goquery.Selection) {
		if found {
			return
		}
		if s.Is("img") == false {
			return
		}
		imgURL, ok := s.Attr("src")
		if ok {
			if strings.HasSuffix(imgURL, ".svg") == false {
				// png gif jpg
				// get imgURL
				imgResp, err := http.Get(imgURL)
				if err != nil {
					log.Println(imgURL, err)
					return
				}
				defer imgResp.Body.Close()

				m, _, err := image.Decode(imgResp.Body)
				if err != nil {
					log.Println(imgURL, err)
					return
				}

				if m.Bounds().Dx() < minWidth || m.Bounds().Dy() < minHeight {
					return
				}
			}

			// use this one
			log.Println("use", imgURL, "for", post)
			result = imgURL
			found = true
		}
	})
	if found {
		return result, nil
	}

	return "", errors.New("can't extract available image from post content")
}

func pickFromBindWallpaper(post string) (string, error) {
	ss := regexPost.FindAllStringSubmatch(post, -1)
	if len(ss) == 0 || len(ss[0]) != 6 {
		return "", errors.New("unexpected parameter")
	}
	// pick from Bing Wallpaper
	tryCount := 1
	tagsCount := len(tags)
	peapixURL := fmt.Sprintf(peapixURLTemplate, ss[0][1], tags[rand.Intn(tagsCount)])
tryAgain:
	peapixResp, err := http.Get(peapixURL)
	if err != nil {
		return "", err
	}
	defer peapixResp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(peapixResp.Body)
	if err != nil {
		return "", err
	}

	found := false
	var result string
	photos := doc.Find(".gallery-photo")
	photoCount := len(photos.Nodes)
	if photoCount == 0 {
		if tryCount == 2 {
			return "", errors.New("can't find proper wall paper")
		}
		tryCount++
		peapixURL = fmt.Sprintf(peapixNoYearURLTemplate, tags[rand.Intn(tagsCount)])
		goto tryAgain
	}
	useIndex := rand.Intn(photoCount)
	photos.Each(func(i int, s *goquery.Selection) {
		if found || i != useIndex {
			return
		}
		imgURL, ok := s.Attr("data-bgset")
		if ok {
			if regexPeapix.MatchString(imgURL) {
				// use this one
				imgURL = strings.Replace(imgURL, "_480.jpg", "_320.jpg", -1)
				log.Println("use", imgURL, "for", post)
				result = imgURL
				found = true
			}
		}
	})

	if found {
		return result, nil
	}

	return "", errors.New("can't pick an available image from Bing Wallpaper")
}

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

	result, err = pickFromBindWallpaper(post)
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

func main() {
	cache = RedisInit()
	rand.Seed(time.Now().Unix())
	bindAddr := os.Getenv("BIND_ADDR")
	if bindAddr == "" {
		bindAddr = "127.0.0.1:8585"
	}
	r := gin.Default()
	r.GET("/:post", handleImageRequestForPost)
	r.DELETE("/:post", handleDeleteCachedPostImage)
	r.POST("/clearall", handleClearAllCachedPostImage)
	log.Fatal(r.Run(bindAddr))
}
