package main

import (
	"bytes"
	"fmt"
	"github.com/anaskhan96/soup"
	"github.com/signintech/gopdf"
	"image"
	"io"
	"net/http"
	"os"
)

func fetchImage(imgLink string) []byte {
	req, e := http.NewRequest("GET", imgLink, nil)
	if e != nil {
		fmt.Println(e)
		os.Exit(1)
	}
	req.Header.Set("Referer", "http://www.webtoons.com")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(response.Body)

	buff := new(bytes.Buffer)
	_, err = buff.ReadFrom(response.Body)
	if err != nil {
		panic(err)
	}
	return buff.Bytes()
}

func main() {
	resp, err := soup.Get("https://www.webtoons.com/en/romance/down-to-earth/s2-episode-169/viewer?title_no=1817&episode_no=169")
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
	println(fmt.Sprintf("Found %d images", len(imgLinks)))

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{Unit: gopdf.UnitPT, PageSize: *gopdf.PageSizeA4})
	for _, imgLink := range imgLinks {
		fmt.Println(imgLink)
		img := fetchImage(imgLink)
		holder, err := gopdf.ImageHolderByBytes(img)
		if err != nil {
			panic(err)
		}

		d, _, err := image.DecodeConfig(bytes.NewReader(img))
		if err != nil {
			panic(err)
		}

		// gopdf assumes dpi 128 https://github.com/signintech/gopdf/issues/168
		// W and H are in points, 1 point = 1/72 inch
		// convert pixels (Width and Height) to ifrom nches, then to points
		// subtract 1 to account for small margins
		pdf.AddPageWithOption(gopdf.PageOption{PageSize: &gopdf.Rect{
			W: float64(d.Width)*72/128 - 1,
			H: float64(d.Height)*72/128 - 1,
		}})
		err = pdf.ImageByHolder(holder, 0, 0, nil)
		if err != nil {
			panic(err)
		}
	}

	err = pdf.WritePdf("out.pdf")
	if err != nil {
		panic(err)
	}
}
