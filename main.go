package main

import (
	"bytes"
	"encoding/json"
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

type MotiontoonJson struct {
	Assets struct {
		Image map[string]string `json:"image"`
	} `json:"assets"`
}

type EpisodeBatch struct {
	imgLinks []string
	minEp    int
	maxEp    int
}

func getOzPageImgLinks(doc soup.Root) []string {
	// regex find the documentURL, e.g:
	// viewerOptions: {
	//        // 필수항목
	//        containerId: '#ozViewer',
	//        documentURL: 'https://global.apis.naver.com/lineWebtoon/webtoon/motiontoonJson.json?seq=2830&hashValue=2e0b924676bdc38241bd8fd452191fe3',
	re := regexp.MustCompile("viewerOptions: \\{\n.*// 필수항목\n.*containerId: '#ozViewer',\n.*documentURL: '(.+)'")
	matches := re.FindStringSubmatch(doc.HTML())
	if len(matches) != 2 {
		fmt.Println("could not find documentURL")
		os.Exit(1)
	}

	// fetch json at documentURL and deserialize to MotiontoonJson
	resp, err := soup.Get(matches[1])
	if err != nil {
		fmt.Println(fmt.Sprintf("Error fetching page: %v", err))
		os.Exit(1)
	}
	var motionToon MotiontoonJson
	if err := json.Unmarshal([]byte(resp), &motionToon); err != nil {
		fmt.Println(fmt.Sprintf("Error unmarshalling json: %v", err))
		os.Exit(1)
	}

	// get sorted keys
	var sortedKeys []string
	for k := range motionToon.Assets.Image {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	// get path rule, e.g:
	// motiontoonParam: {
	//   pathRuleParam: {
	//     stillcut: 'https://ewebtoon-phinf.pstatic.net/motiontoon/3536_2e0b924676bdc38241bd8fd452191fe3/{=filename}?type=q70',
	re = regexp.MustCompile("motiontoonParam: \\{\n.*pathRuleParam: \\{\n.*stillcut: '(.+)'")
	matches = re.FindStringSubmatch(doc.HTML())
	if len(matches) != 2 {
		fmt.Println("could not find pathRule")
		os.Exit(1)
	}
	var imgs []string
	for _, k := range sortedKeys {
		imgs = append(imgs, strings.ReplaceAll(matches[1], "{=filename}", motionToon.Assets.Image[k]))
	}
	return imgs
}

func getImgLinksForEpisode(url string) []string {
	resp, err := soup.Get(url)
	time.Sleep(200 * time.Millisecond)
	if err != nil {
		fmt.Println(fmt.Sprintf("Error fetching page: %v", err))
		os.Exit(1)
	}
	doc := soup.HTMLParse(resp)
	imgs := doc.Find("div", "class", "viewer_lst").FindAll("img")
	if len(imgs) == 0 {
		// some comics seem to serve images from a different backend, something about oz
		return getOzPageImgLinks(doc)
	}
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

func getEpisodeBatches(url string, minEp, maxEp, epsPerBatch int) []EpisodeBatch {
	if strings.Contains(url, "/viewer") {
		// assume viewing single episode
		return []EpisodeBatch{{
			imgLinks: getImgLinksForEpisode(url),
			minEp:    episodeNo(url),
			maxEp:    episodeNo(url),
		}}
	} else {
		// assume viewing set of episodes
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
		actualMinEp := episodeNo(desiredEpisodeLinks[0])
		if minEp > actualMinEp {
			actualMinEp = minEp
		}
		actualMaxEp := episodeNo(desiredEpisodeLinks[len(desiredEpisodeLinks)-1])
		if maxEp < actualMaxEp {
			actualMaxEp = maxEp
		}
		println(fmt.Sprintf("fetching image links for episodes %d through %d", actualMinEp, actualMaxEp))

		var episodeBatches []EpisodeBatch
		for start := 0; start < len(desiredEpisodeLinks); start += epsPerBatch {
			end := start + epsPerBatch
			if end > len(desiredEpisodeLinks) {
				end = len(desiredEpisodeLinks)
			}
			episodeBatches = append(episodeBatches, EpisodeBatch{
				imgLinks: getImgLinksForEpisodes(desiredEpisodeLinks[start:end], actualMaxEp),
				minEp:    episodeNo(desiredEpisodeLinks[start]),
				maxEp:    episodeNo(desiredEpisodeLinks[end-1]),
			})
		}
		return episodeBatches
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

func getImgLinksForEpisodes(episodeLinks []string, actualMaxEp int) []string {
	var allImgLinks []string
	for _, episodeLink := range episodeLinks {
		println(fmt.Sprintf("fetching image links for episode %d/%d", episodeNo(episodeLink), actualMaxEp))
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
	epsPerFile := flag.Int("eps-per-file", 10, "Number of episodes to put in each PDF file")
	flag.Parse()
	if *minEp > *maxEp {
		fmt.Println("min-ep must be less than or equal to max-ep")
		os.Exit(1)
	}
	if *epsPerFile < 1 {
		fmt.Println("eps-per-file must be greater than or equal to 1")
		os.Exit(1)
	}
	if *minEp < 0 {
		fmt.Println("min-ep must be greater than or equal to 0")
		os.Exit(1)
	}

	url := os.Args[len(os.Args)-1]
	episodeBatches := getEpisodeBatches(url, *minEp, *maxEp, *epsPerFile)

	totalPages := 0
	for _, episodeBatch := range episodeBatches {
		totalPages += len(episodeBatch.imgLinks)
	}
	totalEpisodes := episodeBatches[len(episodeBatches)-1].maxEp - episodeBatches[0].minEp + 1
	fmt.Println(fmt.Sprintf("found %d total image links across %d episodes", totalPages, totalEpisodes))
	fmt.Println(fmt.Sprintf("saving into %d files with max of %d episodes per file", len(episodeBatches), *epsPerFile))

	for _, episodeBatch := range episodeBatches {
		pdf := gopdf.GoPdf{}
		pdf.Start(gopdf.Config{Unit: gopdf.UnitPT, PageSize: *gopdf.PageSizeA4})
		for idx, imgLink := range episodeBatch.imgLinks {
			if strings.Contains(imgLink, ".gif") {
				fmt.Println(fmt.Sprintf("WARNING: skipping gif %s", imgLink))
				continue
			}
			err := addImgToPdf(&pdf, imgLink)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			fmt.Println(fmt.Sprintf("saving episodes %d through %d: added page %d/%d", episodeBatch.minEp, episodeBatch.maxEp, idx+1, len(episodeBatch.imgLinks)))
		}

		outURL := strings.ReplaceAll(url, "http://", "")
		outURL = strings.ReplaceAll(outURL, "https://", "")
		outURL = strings.ReplaceAll(outURL, "www.", "")
		outURL = strings.ReplaceAll(outURL, "webtoons.com/", "")
		outURL = strings.Split(outURL, "?")[0]
		outURL = strings.ReplaceAll(outURL, "/viewer", "")
		outURL = strings.ReplaceAll(outURL, "/", "-")
		if episodeBatch.minEp != episodeBatch.maxEp {
			outURL = fmt.Sprintf("%s-epNo%d-epNo%d", outURL, episodeBatch.minEp, episodeBatch.maxEp)
		} else {
			outURL = fmt.Sprintf("%s-epNo%d", outURL, episodeBatch.minEp)
		}
		outPath := outURL + ".pdf"
		err := pdf.WritePdf(outPath)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		fmt.Println(fmt.Sprintf("saved to %s", outPath))
	}
}
