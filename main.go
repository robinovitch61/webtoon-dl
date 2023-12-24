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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func getImgLinksForEpisode(url string) []string {
	resp, err := soup.Get(url)
	time.Sleep(500 * time.Millisecond)
	if err != nil {
		fmt.Println(fmt.Sprintf("Error fetching page: %v", err))
		os.Exit(1)
	}
	doc := soup.HTMLParse(resp)
	imgs := doc.Find("div", "class", "viewer_lst").FindAll("img")

	var imgLinks []string
	for _, img := range imgs {
		if dataURL, ok := img.Attrs()["data-url"]; ok {
			imgLinks = append(imgLinks, dataURL)
		}
	}
	return imgLinks
}

func getEpisodeLinksForPage(url string) ([]string, error) {
	resp, err := soup.Get(url)
	time.Sleep(500 * time.Millisecond)
	if err != nil {
		return []string{}, fmt.Errorf("error fetching page: %v", err)
	}
	doc := soup.HTMLParse(resp)
	episodeURLs := doc.Find("div", "class", "detail_lst").FindAll("a")
	var links []string
	for _, episodeURL := range episodeURLs {
		if href := episodeURL.Attrs()["href"]; strings.Contains(href, "/viewer") {
			links = append(links, href)
		}
	}
	return links, nil
}

func getImgLinks(url string) []string {
	if strings.Contains(url, "/viewer") {
		// assume viewing single episode
		return getImgLinksForEpisode(url)
	} else {
		// assume viewing list of episodes
		re := regexp.MustCompile("&page=[0-9]+")
		allEpisodeLinks := make(map[string]struct{})
		foundLastPage := false
		for page := 1; !foundLastPage; page++ {
			url = re.ReplaceAllString(url, "") + fmt.Sprintf("&page=%d", page)
			episodeLinks, err := getEpisodeLinksForPage(url)
			if err != nil {
				break
			}
			for _, episodeLink := range episodeLinks {
				// when you go past the last page, it just rerenders the last page
				if _, ok := allEpisodeLinks[episodeLink]; ok {
					foundLastPage = true
					break
				}
				allEpisodeLinks[episodeLink] = struct{}{}
			}
			if !foundLastPage {
				println(url)
			}
		}
		keys := make([]string, 0, len(allEpisodeLinks))
		for k := range allEpisodeLinks {
			keys = append(keys, k)
		}
		// extract episode_no from url and sort by it
		re = regexp.MustCompile("episode_no=([0-9]+)")
		episodeNo := func(episodeLink string) int {
			matches := re.FindStringSubmatch(episodeLink)
			if len(matches) != 2 {
				return 0
			}
			episodeNo, err := strconv.Atoi(matches[1])
			if err != nil {
				return 0
			}
			return episodeNo
		}
		sort.Slice(keys, func(i, j int) bool {
			return episodeNo(keys[i]) < episodeNo(keys[j])
		})

		var allImgLinks []string
		for _, episodeLink := range keys {
			println(episodeLink)
			allImgLinks = append(allImgLinks, getImgLinksForEpisode(episodeLink)...)
		}
		return allImgLinks
	}
}

func fetchImage(imgLink string) []byte {
	req, e := http.NewRequest("GET", imgLink, nil)
	if e != nil {
		fmt.Println(e)
		os.Exit(1)
	}
	req.Header.Set("Referer", "http://www.webtoons.com")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	}(response.Body)

	buff := new(bytes.Buffer)
	_, err = buff.ReadFrom(response.Body)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return buff.Bytes()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: webtoon-dl <url>")
		os.Exit(1)
	}
	url := os.Args[1]
	imgLinks := getImgLinks(url)
	fmt.Println(fmt.Sprintf("found %d pages", len(imgLinks)))

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{Unit: gopdf.UnitPT, PageSize: *gopdf.PageSizeA4})
	for _, imgLink := range imgLinks {
		fmt.Println(imgLink)
		img := fetchImage(imgLink)
		holder, err := gopdf.ImageHolderByBytes(img)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		d, _, err := image.DecodeConfig(bytes.NewReader(img))
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		// gopdf assumes dpi 128 https://github.com/signintech/gopdf/issues/168
		// W and H are in points, 1 point = 1/72 inch
		// convert pixels (Width and Height) to points
		// subtract 1 point to account for margins
		pdf.AddPageWithOption(gopdf.PageOption{PageSize: &gopdf.Rect{
			W: float64(d.Width)*72/128 - 1,
			H: float64(d.Height)*72/128 - 1,
		}})
		err = pdf.ImageByHolder(holder, 0, 0, nil)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	}

	outURL := strings.ReplaceAll(url, "http://", "")
	outURL = strings.ReplaceAll(outURL, "https://", "")
	outURL = strings.ReplaceAll(outURL, "www.", "")
	outURL = strings.ReplaceAll(outURL, "webtoons.com/", "")
	outURL = strings.Split(outURL, "?")[0]
	outURL = strings.ReplaceAll(outURL, "/viewer", "")
	outURL = strings.ReplaceAll(outURL, "/", "-")
	outPath := outURL + ".pdf"
	err := pdf.WritePdf(outPath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Println(fmt.Sprintf("saved to %s", outPath))
}
