package main

import (
	"fmt"
	"github.com/anaskhan96/soup"
	"os"
)

func main() {
	resp, err := soup.Get("https://www.webtoons.com/en/thriller/bastard/ep-0/viewer?title_no=485&episode_no=1")
	if err != nil {
		fmt.Println(fmt.Sprintf("Error fetching page: %v", err))
		os.Exit(1)
	}
	doc := soup.HTMLParse(resp)
	imgs := doc.Find("div", "class", "viewer_lst").FindAll("img")
	var imgLinks []string
	for _, img := range imgs {
		imgLinks = append(imgLinks, img.Attrs()["data-url"])
	}
	println(imgLinks[0])
}
