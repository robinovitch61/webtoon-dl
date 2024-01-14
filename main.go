package main

import (
	"archive/zip"
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

type EpisodeBatch struct {
	imgLinks []string
	minEp    int
	maxEp    int
}

type OutputFile interface {
	addImg(imgLink string) error
	save() error
}

type PdfFile struct {
	pdf     *gopdf.GoPdf
	outPath string
}

func NewPdfFile(outPath string) *PdfFile {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{Unit: gopdf.UnitPT, PageSize: *gopdf.PageSizeA4})
	return &PdfFile{
		pdf:     &pdf,
		outPath: outPath,
	}
}

func (pf *PdfFile) addImg(imgLink string) error {
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
	pf.pdf.AddPageWithOption(gopdf.PageOption{PageSize: &gopdf.Rect{
		W: float64(d.Width)*72/128 - 1,
		H: float64(d.Height)*72/128 - 1,
	}})
	return pf.pdf.ImageByHolder(holder, 0, 0, nil)
}

func (pf *PdfFile) save() error {
	return pf.pdf.WritePdf(pf.outPath)
}

type CbzFile struct {
	pngs    []byte
	outPath  string
}

func NewCbzFile(outPath string) *CbzFile {
	return &CbzFile{
		pngs
		outPath: outPath,
	}
}

func (cf *CbzFile) addImg(imgLink string) error {
	img := fetchImage(imgLink)

}

func (cf *CbzFile) save() error {
	zipFile, err := os.Create(cf.outPath)
	if err != nil {
		return err
	}
	defer func(zipFile *os.File) {
		err := zipFile.Close()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	}(zipFile)

	zipWriter := NewZipWriter(zipFile)
	defer func(zipWriter *ZipWriter) {
		err := zipWriter.Close()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	}(zipWriter)

	for _, imgLink := range cf.imgLinks {
		img := fetchImage(imgLink)
		_, err := zipWriter.Write(img)
		if err != nil {
			return err
		}
	}
	return nil
}

type ZipWriter struct {
	*zip.Writer
}

func NewZipWriter(w io.Writer) *ZipWriter {
	return &ZipWriter{zip.NewWriter(w)}
}

func (zw *ZipWriter) Write(data []byte) (int, error) {
	f, err := zw.CreateHeader(&zip.FileHeader{
		Name:   strconv.Itoa(time.Now().Nanosecond()),
		Method: zip.Store,
	})
	if err != nil {
		return 0, err
	}
	return f.Write(data)
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

func getOutPath(url string, minEp, maxEp int, outputFormat string) string {
	outURL := strings.ReplaceAll(url, "http://", "")
	outURL = strings.ReplaceAll(outURL, "https://", "")
	outURL = strings.ReplaceAll(outURL, "www.", "")
	outURL = strings.ReplaceAll(outURL, "webtoons.com/", "")
	outURL = strings.Split(outURL, "?")[0]
	outURL = strings.ReplaceAll(outURL, "/viewer", "")
	outURL = strings.ReplaceAll(outURL, "/", "-")
	if minEp != maxEp {
		outURL = fmt.Sprintf("%s-epNo%d-epNo%d", outURL, minEp, maxEp)
	} else {
		outURL = fmt.Sprintf("%s-epNo%d", outURL, minEp)
	}
	if outputFormat == "pdf" {
		outURL += ".pdf"
	} else {
		outURL += ".cbz"
	}
	return outURL
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: webtoon-dl <url>")
		os.Exit(1)
	}
	minEp := flag.Int("min-ep", 0, "Minimum episode number to download (inclusive)")
	maxEp := flag.Int("max-ep", math.MaxInt, "Maximum episode number to download (inclusive)")
	epsPerFile := flag.Int("eps-per-file", 10, "Number of episodes to put in each PDF file. Default 10")
	outputFormat := flag.String("fmt", "pdf", "Output format (pdf or cbz). Default pdf")
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
	if *outputFormat != "pdf" && *outputFormat != "cbz" {
		fmt.Println("fmt must be pdf or cbz")
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
		var output OutputFile
		outputPath := getOutPath(url, episodeBatch.minEp, episodeBatch.maxEp, *outputFormat)
		if *outputFormat == "pdf" {
			output = NewPdfFile(outputPath)
		} else {
			output = NewCbzFile(outputPath)
		}
		for idx, imgLink := range episodeBatch.imgLinks {
			err := output.addImg(imgLink)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			fmt.Println(fmt.Sprintf("saving episodes %d through %d: added page %d/%d", episodeBatch.minEp, episodeBatch.maxEp, idx+1, len(episodeBatch.imgLinks)))
		}

		err := output.save()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		fmt.Println(fmt.Sprintf("saved to %s", outputPath))
	}
}
