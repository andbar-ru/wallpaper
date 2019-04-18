/*
Program downloads and sets random wallpaper from wallhaven.cc according to command-line arguments.
*/
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/andbar-ru/average_color"
)

const (
	BASE_URL   = "https://alpha.wallhaven.cc"
	CATEGORIES = "100" // +General,-Anime,-People
	PURITY     = "100" // +SWF(safe for work),-Sketchy,?
	SORTING    = "random"

	USER_AGENT = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.122 Safari/537.36"

	hDesc = "print this help"
	rDesc = "download and set random wallpaper without comparing with check color"
)

var (
	// Set from сommand line arguments.
	checkColor *color.NRGBA

	imagesDir  string
	resolution string
	client     = &http.Client{}
	colorRgx   = regexp.MustCompile(`^#?[0-9a-fA-F]{6}$`)
)

func check(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func printHelpAndExit(code int) {
	fmt.Printf("Usage: %s [flags] <color>\n\n", os.Args[0])
	fmt.Println("Color is given in format 'rrggbb' or '#rrggbb'.\n")
	fmt.Println("Flags:")
	fmt.Printf("  -h  %s\n", hDesc)
	fmt.Printf("  -r  %s\n", rDesc)

	os.Exit(code)
}

func parseArgs() {
	if len(os.Args) == 1 {
		printHelpAndExit(0)
	}

	h := flag.Bool("h", false, hDesc)
	r := flag.Bool("r", false, rDesc)

	flag.Parse()

	if *h {
		printHelpAndExit(0)
	}
	if *r {
		return
	}

	colorStr := flag.Arg(0)
	if colorStr == "" {
		fmt.Println("ERROR: color is not specified.\n")
		printHelpAndExit(1)
	}
	if !colorRgx.MatchString(colorStr) {
		fmt.Println("ERROR: color is in wrong format.\n")
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

func setResolutions() {
	cmd := "xdpyinfo | awk '/dimensions/{print $2}'"
	out, err := exec.Command("bash", "-c", cmd).Output()
	check(err)
	resolution = strings.TrimSpace(string(out))
}

func setImagesDir() {
	imagesDir = fmt.Sprintf("%s/Images/%s", os.Getenv("HOME"), resolution)
	// Create directory if not exists.
	_, err := os.Stat(imagesDir)
	if os.IsNotExist(err) {
		err = os.Mkdir(imagesDir, 0755)
		check(err)
	}
}

func getResponse(url string) *http.Response {
	request, err := http.NewRequest("GET", url, nil)
	check(err)
	request.Header.Set("User-Agent", USER_AGENT)
	response, err := client.Do(request)
	check(err)
	if response.StatusCode != 200 {
		log.Panicf("Status code error: %d %s", response.StatusCode, response.Status)
	}
	return response
}

// getDocument returns goquery Document from page content on url.
func getDocument(url string) *goquery.Document {
	response := getResponse(url)
	defer response.Body.Close()
	document, err := goquery.NewDocumentFromReader(response.Body)
	check(err)
	return document
}

// getColorDistance returns Euclidean distance between two NRGBA colors.
func getColorDistance(c1, c2 color.NRGBA) float64 {
	r1 := float64(c1.R)
	r2 := float64(c2.R)
	g1 := float64(c1.G)
	g2 := float64(c2.G)
	b1 := float64(c1.B)
	b2 := float64(c2.B)
	a1 := float64(c1.A)
	a2 := float64(c2.A)

	return math.Sqrt((r1-r2)*(r1-r2) + (g1-g2)*(g1-g2) + (b1-b2)*(b1-b2) + (a1-a2)*(a1-a2))
}

// pickThumb picks thumb which closest to color. If not color, picks first thumb.
// Returns href to the next page.
func pickThumb(thumbs *goquery.Selection) *goquery.Selection {
	var thumb *goquery.Selection
	if checkColor == nil {
		thumb = thumbs.First()
	} else {
		minDistance := 500.0

		thumbs.Each(func(i int, th *goquery.Selection) {
			src, ok := th.Find("img").Attr("data-src")
			if !ok {
				log.Panic("Could not find thumb src")
			}
			response := getResponse(src)
			defer response.Body.Close()
			img, _, err := image.Decode(response.Body)
			check(err)
			avgColor := average_color.AverageColor(img)
			distance := getColorDistance(avgColor, *checkColor)
			if distance < minDistance {
				minDistance = distance
				thumb = th
			}
		})
	}
	return thumb
}

// downloadImage downloads image to imagesDir and returns path to it.
func downloadImage(src string) string {
	imagePath := path.Join(imagesDir, path.Base(src))
	output, err := os.Create(imagePath)
	if err != nil {
		log.Panicf("Could not create file %s, err: %s", imagePath, err)
	}
	defer output.Close()
	response := getResponse(src)
	defer response.Body.Close()
	_, err = io.Copy(output, response.Body)
	if err != nil {
		log.Panicf("Could not write image to file, err: %s", err)
	}
	return imagePath
}

// Set wallpaper.
func setWallpaper(imagePath string) {
	cmd := exec.Command("fbsetbg", "-f", imagePath)
	err := cmd.Run()
	check(err)
}

func main() {
	parseArgs()

	setResolutions()
	setImagesDir()

	url := fmt.Sprintf("%s/search?categories=%s&purity=%s&resolutions=%s&sorting=%s", BASE_URL, CATEGORIES, PURITY, resolution, SORTING)

	// Page with thumbs.
	page := getDocument(url)
	thumbs := page.Find("figure.thumb")
	if thumbs.Length() == 0 {
		log.Panicf("Could not find thumbs")
	}

	thumb := pickThumb(thumbs)
	href, ok := thumb.Find(".preview").Attr("href")
	if !ok {
		log.Panic("Could not find thumb's preview href")
	}

	// Preview page.
	page = getDocument(href)
	img := page.Find("#wallpaper")
	src, ok := img.Attr("src")
	if !ok {
		log.Panic("Could not find wallpaper's src on preview page")
	}
	if !strings.HasPrefix(src, "http") {
		src = "https:" + src
	}

	fmt.Println(src)
	imagePath := downloadImage(src)
	setWallpaper(imagePath)
}
