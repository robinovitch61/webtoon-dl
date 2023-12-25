package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/anaskhan96/soup"
	"github.com/signintech/gopdf"
	"image"
	"io"
	"math"
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
	time.Sleep(200 * time.Millisecond)
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
	time.Sleep(200 * time.Millisecond)
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

func getImgLinks(url string, minEp, maxEp int) ([]string, int, int) {
	if strings.Contains(url, "/viewer") {
		// assume viewing single episode
		return getImgLinksForEpisode(url), episodeNo(url), episodeNo(url)
	} else {
		// assume viewing list of episodes
		println("scanning all pages to get all episode links")
		allEpisodeLinks := getAllEpisodeLinks(url)
		println(fmt.Sprintf("found %d total episodes", len(allEpisodeLinks)))

		var desiredEpisodeLinks []string
		for _, episodeLink := range allEpisodeLinks {
			epNo := episodeNo(episodeLink)
			if epNo >= minEp && epNo <= maxEp {
				desiredEpisodeLinks = append(desiredEpisodeLinks, episodeLink)
			}
		}

		return getImgLinksForEpisodes(desiredEpisodeLinks), episodeNo(desiredEpisodeLinks[0]), episodeNo(desiredEpisodeLinks[len(desiredEpisodeLinks)-1])
	}
}

func getAllEpisodeLinks(url string) []string {
	re := regexp.MustCompile("&page=[0-9]+")
	episodeLinkSet := make(map[string]struct{})
	foundLastPage := false
	for page := 1; !foundLastPage; page++ {
		url = re.ReplaceAllString(url, "") + fmt.Sprintf("&page=%d", page)
		episodeLinks, err := getEpisodeLinksForPage(url)
		if err != nil {
			break
		}
		for _, episodeLink := range episodeLinks {
			// when you go past the last page, it just rerenders the last page
			if _, ok := episodeLinkSet[episodeLink]; ok {
				foundLastPage = true
				break
			}
			episodeLinkSet[episodeLink] = struct{}{}
		}
		if !foundLastPage {
			println(url)
		}
	}

	allEpisodeLinks := make([]string, 0, len(episodeLinkSet))
	for episodeLink := range episodeLinkSet {
		allEpisodeLinks = append(allEpisodeLinks, episodeLink)
	}

	// extract episode_no from url and sort by it
	sort.Slice(allEpisodeLinks, func(i, j int) bool {
		return episodeNo(allEpisodeLinks[i]) < episodeNo(allEpisodeLinks[j])
	})
	return allEpisodeLinks
}

func episodeNo(episodeLink string) int {
	re := regexp.MustCompile("episode_no=([0-9]+)")
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

func getImgLinksForEpisodes(episodeLinks []string) []string {
	var allImgLinks []string
	for _, episodeLink := range episodeLinks {
		println(fmt.Sprintf("fetching images for episode %d (last episode %d)", episodeNo(episodeLink), episodeNo(episodeLinks[len(episodeLinks)-1])))
		allImgLinks = append(allImgLinks, getImgLinksForEpisode(episodeLink)...)
	}
	return allImgLinks
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

func addImgToPdf(pdf *gopdf.GoPdf, imgLink string) error {
	img := fetchImage(imgLink)
	holder, err := gopdf.ImageHolderByBytes(img)
	if err != nil {
		return err
	}

	d, _, err := image.DecodeConfig(bytes.NewReader(img))
	if err != nil {
		return err
	}

	// gopdf assumes dpi 128 https://github.com/signintech/gopdf/issues/168
	// W and H are in points, 1 point = 1/72 inch
	// convert pixels (Width and Height) to points
	// subtract 1 point to account for margins
	pdf.AddPageWithOption(gopdf.PageOption{PageSize: &gopdf.Rect{
		W: float64(d.Width)*72/128 - 1,
		H: float64(d.Height)*72/128 - 1,
	}})
	return pdf.ImageByHolder(holder, 0, 0, nil)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: webtoon-dl <url>")
		os.Exit(1)
	}
	minEp := flag.Int("min-ep", 0, "Minimum episode number to download (inclusive)")
	maxEp := flag.Int("max-ep", math.MaxInt, "Maximum episode number to download (inclusive)")
	flag.Parse()
	if *minEp > *maxEp {
		fmt.Println("min-ep must be less than or equal to max-ep")
		os.Exit(1)
	}

	url := os.Args[len(os.Args)-1]
	imgLinks, actualMinEp, actualMaxEp := getImgLinks(url, *minEp, *maxEp)
	fmt.Println(fmt.Sprintf("found %d pages", len(imgLinks)))

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{Unit: gopdf.UnitPT, PageSize: *gopdf.PageSizeA4})
	for idx, imgLink := range imgLinks {
		err := addImgToPdf(&pdf, imgLink)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		fmt.Println(fmt.Sprintf("added page %d/%d", idx+1, len(imgLinks)))
	}

	outURL := strings.ReplaceAll(url, "http://", "")
	outURL = strings.ReplaceAll(outURL, "https://", "")
	outURL = strings.ReplaceAll(outURL, "www.", "")
	outURL = strings.ReplaceAll(outURL, "webtoons.com/", "")
	outURL = strings.Split(outURL, "?")[0]
	outURL = strings.ReplaceAll(outURL, "/viewer", "")
	outURL = strings.ReplaceAll(outURL, "/", "-")
	if actualMinEp != actualMaxEp {
		outURL = fmt.Sprintf("%s-ep%d-%d", outURL, actualMinEp, actualMaxEp)
	} else {
		outURL = fmt.Sprintf("%s-ep%d", outURL, actualMinEp)
	}
	outPath := outURL + ".pdf"
	err := pdf.WritePdf(outPath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Println(fmt.Sprintf("saved to %s", outPath))
}
