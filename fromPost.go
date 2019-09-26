package main

import (
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	postPattern = `^(\d{4})\-(\d{2})\-(\d{2})\-([^\.]+)\.(html|md)$`
	minWidth    = 480
	minHeight   = 320
)

var (
	regexPost = regexp.MustCompile(postPattern)
)

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
