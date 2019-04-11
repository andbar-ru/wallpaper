/*
Program downloads and sets random wallpaper from wallhaven.cc according to command-line arguments.
*/
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	BASE_URL   = "https://alpha.wallhaven.cc"
	CATEGORIES = "100" // +General,-Anime,-People
	PURITY     = "100" // +SWF(safe for work),-Sketchy,?
	SORTING    = "random"

	USER_AGENT = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.122 Safari/537.36"
)

var (
	imagesDir  string
	resolution string
	client     = &http.Client{}
)

func check(err error) {
	if err != nil {
		log.Panic(err)
	}
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

// getRoot returns goquery Document from page content on url.
func getDocument(url string) *goquery.Document {
	request, err := http.NewRequest("GET", url, nil)
	check(err)
	request.Header.Set("User-Agent", USER_AGENT)
	response, err := client.Do(request)
	check(err)
	defer response.Body.Close()
	if response.StatusCode != 200 {
		log.Panicf("status code error: %d %s", response.StatusCode, response.Status)
	}
	document, err := goquery.NewDocumentFromReader(response.Body)
	check(err)
	return document
}

// pickThumb picks thumb which closest to color. If not color, picks first thumb.
// Returns href to the next page.
func pickThumb(thumbs *goquery.Selection) string {
	thumb := thumbs.First()
	href, ok := thumb.Find(".preview").Attr("href")
	if !ok {
		log.Panic("pickThumb: Could not find thumb's preview href")
	}
	return href
}

// downloadImage downloads image to imagesDir and returns path to it.
func downloadImage(src string) string {
	imagePath := path.Join(imagesDir, path.Base(src))
	output, err := os.Create(imagePath)
	if err != nil {
		log.Panicf("Could not create file %s, err: %s", imagePath, err)
	}
	defer output.Close()
	request, err := http.NewRequest("GET", src, nil)
	check(err)
	request.Header.Set("User-Agent", USER_AGENT)
	response, err := client.Do(request)
	if err != nil {
		log.Panicf("Could not download image from %s, err: %s", src, err)
	}
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
	setResolutions()
	setImagesDir()

	url := fmt.Sprintf("%s/search?categories=%s&purity=%s&resolutions=%s&sorting=%s", BASE_URL, CATEGORIES, PURITY, resolution, SORTING)

	// Page with thumbs.
	page := getDocument(url)
	thumbs := page.Find("figure.thumb")
	if thumbs.Length() == 0 {
		log.Panicf("Could not find thumbs")
	}

	// Preview page.
	href := pickThumb(thumbs)
	page = getDocument(href)
	img := page.Find("#wallpaper")
	src, ok := img.Attr("src")
	if !ok {
		log.Panic("pickThumb: Could not find wallpaper's src on preview page")
	}
	if !strings.HasPrefix(src, "http") {
		src = "https:" + src
	}

	imagePath := downloadImage(src)
	setWallpaper(imagePath)
}
