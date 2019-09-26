package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gomodule/redigo/redis"
)

const (
	peapixURLTemplate     = `https://peapix.com/bing?year=%s`
	peapixPageURLTemplate = `https://peapix.com/bing?year=%s&page=%d`
	peapixPattern         = `^https:\/\/img\.peapix\.com\/[0-9a-z]+_[0-9]{3}\.jpg$`
)

var (
	regexPeapix = regexp.MustCompile(peapixPattern)
)

func extractImageURLs(doc *goquery.Document) (res []string) {
	doc.Find(".gallery-photo").Each(func(i int, s *goquery.Selection) {
		imgURL, ok := s.Attr("data-bgset")
		if ok {
			if regexPeapix.MatchString(imgURL) {
				// use this one
				imgURL = strings.Replace(imgURL, "_480.jpg", "_320.jpg", -1)
				res = append(res, imgURL)
			}
		}
	})

	return
}

// pickFromBingWallpaper work flow:
// check redis first
// if there's no records in redis, extract all image URLs from peapix
// save all URLs into redis for 24 hours
// randomly select an image from the set and return
func pickFromBingWallpaper(post string) (string, error) {
	ss := regexPost.FindAllStringSubmatch(post, -1)
	if len(ss) == 0 || len(ss[0]) != 6 {
		return "", errors.New("unexpected parameter")
	}

	peapixURL := fmt.Sprintf(peapixURLTemplate, ss[0][1])

	if cache.IsExist(peapixURL) {
		imgURL, err := redis.String(cache.RandSetMember(peapixURL))
		if err == nil {
			return imgURL, nil
		}
	}

	// query all pages
	peapixResp, err := http.Get(peapixURL)
	if err != nil {
		return "", err
	}
	defer peapixResp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(peapixResp.Body)
	if err != nil {
		return "", err
	}

	maxPage := 1
	doc.Find(".page-link").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if pageNo, err := strconv.Atoi(text); err == nil {
			if pageNo > maxPage {
				maxPage = pageNo
			}
		}
	})

	// query all images from all pages
	res := extractImageURLs(doc)

	for i := maxPage; i > 1; i-- {
		u := fmt.Sprintf(peapixPageURLTemplate, ss[0][1], i)
		resp, err := http.Get(u)
		if err != nil {
			continue
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err == nil {
			r := extractImageURLs(doc)
			res = append(res, r...)
		}
		resp.Body.Close()
	}

	count := len(res)
	if count > 0 {
		s := make([]interface{}, count)
		for i, v := range res {
			s[i] = v
		}
		if _, err = cache.SetSet(peapixURL, s...); err != nil {
			log.Println(err)
		}
		return res[rand.Intn(count)], nil
	}

	return "", errors.New("can't pick an available image from Bing Wallpaper")
}
