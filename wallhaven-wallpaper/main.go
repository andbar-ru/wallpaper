/*
Program downloads and sets random wallpaper for current screen resolution from wallhaven.cc
according to command-line arguments.
*/
package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	netUrl "net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"wallpaper/utils"

	"github.com/PuerkitoBio/goquery"
	"github.com/andbar-ru/average_color"
)

const (
	baseURL    = "https://wallhaven.cc"
	categories = "100" // +General,-Anime,-People
	purity     = "100" // +SWF(safe for work),-Sketchy,?
	sorting    = "random"

	maxDistance = 500.0
	userAgent   = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.122 Safari/537.36"

	hDesc   = "print this help"
	rDesc   = "true random - download and set random wallpaper without comparing with check color"
	tDesc   = "threshold - maximum allowed distance between thumb average color and check color. Must be between 0 and 500."
	lDesc   = "last page to process. In the case of random search this is the number of tryings as next pages can have duplicate thumbs."
	symbols = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 !\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
)

var (
	// Set from сommand line arguments.
	checkColor *color.NRGBA
	threshold  float64
	lastPage   int

	imagesDir  string
	resolution string
	client     = newClient()
	colorRgx   = regexp.MustCompile(`^#?[0-9a-fA-F]{6}$`)
	// Start page. While distance greater than threshold, go to the next page.
	page          = 1
	pageHeaderRgx = regexp.MustCompile(`\d+\s*/\s*(\d+)`)
	seed          = getSeed()
)

// Custom http.Client.
type customClient struct {
	client  *http.Client
	referer string
}

func newClient() *customClient {
	// There is self-signed sertificate in Kronshtadt Technologies.
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &customClient{client: &http.Client{Transport: transport}}
	return client
}

func (client *customClient) get(url string) *http.Response {
	urlParsed, err := netUrl.Parse(url)
	check(err)
	urlPath := urlParsed.Path

	request, err := http.NewRequest("GET", url, nil)
	check(err)

	request.Header.Set("user-agent", userAgent)
	if client.referer != "" {
		request.Header.Set("referer", client.referer)
	}
	if urlPath == "/search" {
		request.Header.Set("x-requested-with", "XMLHttpRequest")
	}

	response, err := client.client.Do(request)
	check(err)
	if response.StatusCode != 200 {
		log.Panicf("Status code error: %d %s", response.StatusCode, response.Status)
	}

	if urlPath == "/search" {
		client.referer = url
	}
	if client.client.Jar == nil {
		jar, err := cookiejar.New(nil)
		check(err)
		u, err := netUrl.Parse(baseURL)
		check(err)
		jar.SetCookies(u, response.Cookies())
		client.client.Jar = jar
	}

	return response
}

// getSeed returns random strings which will be used as a seed.
func getSeed() string {
	var s string
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 16; i++ {
		s += string(symbols[random.Intn(len(symbols))])
	}
	return netUrl.QueryEscape(s)
}

func check(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func printHelpAndExit(code int) {
	fmt.Printf("Usage: %s [flags] <color>\n\n", os.Args[0])
	fmt.Print("Color is given in format 'rrggbb' or '#rrggbb'.\n\n")
	fmt.Println("Flags:")
	fmt.Printf("  -h  %s\n", hDesc)
	fmt.Printf("  -r  %s\n", rDesc)
	fmt.Printf("  -t float  %s\n", tDesc)
	fmt.Printf("  -l int  %s\n", lDesc)

	os.Exit(code)
}

func parseArgs() {
	if len(os.Args) == 1 {
		printHelpAndExit(0)
	}

	h := flag.Bool("h", false, hDesc)
	r := flag.Bool("r", false, rDesc)
	flag.Float64Var(&threshold, "t", maxDistance, tDesc)
	flag.IntVar(&lastPage, "l", 0, lDesc)

	flag.Parse()

	if *h {
		printHelpAndExit(0)
	}
	if *r {
		return
	}
	if threshold <= 0 {
		fmt.Print("ERROR: threshold must be positive.\n\n")
		printHelpAndExit(1)
	}
	if threshold > maxDistance {
		threshold = maxDistance
	}

	colorStr := flag.Arg(0)
	if colorStr == "" {
		fmt.Print("ERROR: color is not specified.\n\n")
		printHelpAndExit(1)
	}
	if !colorRgx.MatchString(colorStr) {
		fmt.Print("ERROR: color is in wrong format.\n\n")
		printHelpAndExit(1)
	}
	if colorStr[0] == '#' {
		colorStr = colorStr[1:]
	}
	red, err := strconv.ParseUint(colorStr[0:2], 16, 8)
	check(err)
	green, err := strconv.ParseUint(colorStr[2:4], 16, 8)
	check(err)
	blue, err := strconv.ParseUint(colorStr[4:6], 16, 8)
	check(err)
	checkColor = &color.NRGBA{uint8(red), uint8(green), uint8(blue), 0xff}
}

// setResolution finds out screen resolution and sets variable.
func setResolution() {
	cmd := "xdpyinfo | awk '/dimensions/{print $2}'"
	out, err := exec.Command("bash", "-c", cmd).Output()
	check(err)
	resolution = strings.TrimSpace(string(out))
}

// setImagesDir sets variable according to current user and screen resolution.
func setImagesDir() {
	imagesDir = fmt.Sprintf("%s/Images/%s", os.Getenv("HOME"), resolution)
	// Create directory if not exists.
	_, err := os.Stat(imagesDir)
	if os.IsNotExist(err) {
		err = os.Mkdir(imagesDir, 0755)
		check(err)
	}
}

// getDocument returns goquery Document from page content on url.
func getDocument(url string) *goquery.Document {
	response := client.get(url)
	defer response.Body.Close()
	document, err := goquery.NewDocumentFromReader(response.Body)
	check(err)
	return document
}

// pickThumb picks thumb which closest to color and returns it with thumb's average color and
// distance from the checkColor. If not color, returns first thumb.
func pickThumb(thumbs *goquery.Selection) (*goquery.Selection, *color.NRGBA, float64) {
	if checkColor == nil {
		return thumbs.First(), nil, -1
	}

	type result struct {
		thumb    *goquery.Selection
		avgColor color.NRGBA
		distance float64
	}

	var (
		closestThumb             *goquery.Selection
		closestThumbAverageColor *color.NRGBA
		minDistance              = maxDistance
		results                  = make(chan result, thumbs.Length())
	)

	thumbs.Each(func(i int, thumb *goquery.Selection) {
		go func() {
			src, ok := thumb.Find("img").Attr("data-src")
			if !ok {
				log.Panic("Could not find thumb src")
			}
			response := client.get(src)
			defer response.Body.Close()
			img, _, err := image.Decode(response.Body)
			if err != nil {
				// Pass image
				log.Printf("ERROR: %s: %s", src, err)
				results <- result{thumb, color.NRGBA{}, maxDistance}
				return
			}
			avgColor := average_color.AverageColor(img)
			distance := utils.GetColorDistance(&avgColor, checkColor)
			results <- result{thumb, avgColor, distance}
		}()
	})

	for i := 0; i < thumbs.Length(); i++ {
		res := <-results
		if res.distance < minDistance {
			minDistance = res.distance
			closestThumb = res.thumb
			closestThumbAverageColor = &res.avgColor
		}
	}

	return closestThumb, closestThumbAverageColor, minDistance
}

// downloadImage downloads image to imagesDir and returns path to it.
func downloadImage(src string) string {
	imagePath := path.Join(imagesDir, path.Base(src))
	output, err := os.Create(imagePath)
	if err != nil {
		log.Panicf("Could not create file %s, err: %s", imagePath, err)
	}
	defer output.Close()
	response := client.get(src)
	defer response.Body.Close()
	_, err = io.Copy(output, response.Body)
	if err != nil {
		log.Panicf("Could not write image to file, err: %s", err)
	}
	return imagePath
}

func main() {
	parseArgs()

	setResolution()
	setImagesDir()

	var thumb, closestThumb *goquery.Selection
	var avgColor, closestAvgColor *color.NRGBA
	distance := threshold + 1
	closestDistance := maxDistance
	var pageOf int

	for distance > threshold {
		url := fmt.Sprintf("%s/search?categories=%s&purity=%s&resolutions=%s&sorting=%s&seed=%s&page=%d", baseURL, categories, purity, resolution, sorting, seed, page)

		// Page with thumbs.
		doc := getDocument(url)

		// Setting lastPage and pageOf.
		// We can't set these variables on the first page because page info becomes available since second page.
		if page == 2 {
			pageHeader := doc.Find(".thumb-listing-page-header").Text()
			submatches := pageHeaderRgx.FindStringSubmatch(pageHeader)
			var err error
			pageOf, err = strconv.Atoi(submatches[1])
			check(err)
			if lastPage < 1 || lastPage > pageOf {
				lastPage = pageOf
			}
		}

		thumbs := doc.Find("figure.thumb")
		if thumbs.Length() == 0 {
			log.Panicf("Could not find thumbs")
		}

		if page == 1 {
			if checkColor != nil {
				fmt.Printf("Picking a thumb out of %d which has average color closest to %s...\n", thumbs.Length(), utils.Color2hex(checkColor))
			} else {
				fmt.Printf("Picking first thumb out of %d.\n", thumbs.Length())
			}
		}

		thumb, avgColor, distance = pickThumb(thumbs)

		if distance > threshold {
			if distance < closestDistance {
				closestThumb = thumb
				closestAvgColor = avgColor
				closestDistance = distance
			}
			page++
			// If we have reached last page but could not find appropriate thumb, accept closest found.
			if page > lastPage && lastPage != 0 {
				fmt.Println("Could not find appropriate thumb, picking the closest one")
				thumb = closestThumb
				avgColor = closestAvgColor
				distance = closestDistance
				break
			}
			if pageOf != 0 {
				fmt.Printf("%.2f > %.2f go to page %d of %d\n", distance, threshold, page, pageOf)
			} else {
				fmt.Printf("%.2f > %.2f go to page %d of ?\n", distance, threshold, page)
			}
		}
	}

	href, ok := thumb.Find(".preview").Attr("href")
	if !ok {
		log.Panic("Could not find thumb's preview href")
	}

	if avgColor != nil {
		fmt.Printf("Result: average color %s, distance %.2f\n", utils.Color2hex(avgColor), distance)
	}

	// Preview page.
	doc := getDocument(href)
	img := doc.Find("#wallpaper")
	src, ok := img.Attr("src")
	if !ok {
		log.Panic("Could not find wallpaper's src on preview page")
	}
	if !strings.HasPrefix(src, "http") {
		src = "https:" + src
	}

	fmt.Println(src)
	imagePath := downloadImage(src)
	err := utils.SetWallpaper(imagePath)
	check(err)
}
