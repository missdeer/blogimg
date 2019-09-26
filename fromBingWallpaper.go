package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	peapixURLTemplate = `https://peapix.com/bing?year=%s`
	peapixPattern     = `^https:\/\/img\.peapix\.com\/[0-9a-z]+_[0-9]{3}\.jpg$`
)

var (
	regexPeapix = regexp.MustCompile(peapixPattern)
)

func pickFromBingWallpaper(post string) (string, error) {
	ss := regexPost.FindAllStringSubmatch(post, -1)
	if len(ss) == 0 || len(ss[0]) != 6 {
		return "", errors.New("unexpected parameter")
	}
	// pick from Bing Wallpaper
	peapixURL := fmt.Sprintf(peapixURLTemplate, ss[0][1])

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
		return "", errors.New("can't find proper wall paper")
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
