/*
This utility sets wallpaper from directory specified as cli argument. Wallpaper can be random or
closest to the specified color accorinding to cli arguments.
*/
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	_ "image/jpeg"
	_ "image/png"

	"github.com/andbar-ru/average_color"

	"wallpaper/logger"
	"wallpaper/utils"
)

const (
	hDesc = "print this help"
	dDesc = "directory to take wallpaper from"
	rDesc = "set random wallpaper"
	mDesc = "set max number of images to process to avoid continuous running"
)

var (
	dir        string
	checkColor *color.NRGBA
	maxItems   int

	colorRgx        = regexp.MustCompile(`^#?[0-9a-fA-F]{6}$`)
	log             = logger.NewConsoleLogger(0)
	validExtensions = map[string]struct{}{
		".png":  {},
		".jpg":  {},
		".jpeg": {},
	}
)

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
	fmt.Printf("  -d  %s\n", dDesc)
	fmt.Printf("  -r  %s\n", rDesc)
	fmt.Printf("  -m  %s\n", mDesc)

	os.Exit(code)
}

func parseArgs() {
	if len(os.Args) == 1 {
		printHelpAndExit(0)
	}

	h := flag.Bool("h", false, hDesc)
	r := flag.Bool("r", false, rDesc)
	flag.StringVar(&dir, "d", "", dDesc)
	flag.IntVar(&maxItems, "m", 0, mDesc)

	flag.Parse()

	if *h {
		printHelpAndExit(0)
	}
	if dir == "" {
		log.Error("Image directory is not specified.\n")
		printHelpAndExit(1)
	}
	fileInfo, err := os.Stat(dir)
	check(err)
	if !fileInfo.IsDir() {
		log.Error(fmt.Sprintf("%s is not a directory\n", dir))
		printHelpAndExit(1)
	}
	if *r {
		if flag.Arg(0) != "" {
			log.Warn("Randomness flag and color are both specified. Randomness flag has higher priority.")
		}
		return
	}

	colorStr := flag.Arg(0)
	if colorStr == "" {
		log.Error("Color is not specified.\n")
		printHelpAndExit(1)
	}
	if !colorRgx.MatchString(colorStr) {
		log.Error("Color is in wrong format.\n")
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

func main() {
	parseArgs()

	files := make([]string, 0)
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Panic(err)
		}
		ext := strings.ToLower(filepath.Ext(path))
		if _, ok := validExtensions[ext]; ok {
			files = append(files, path)
		}
		return nil
	})
	if len(files) == 0 {
		log.Error(fmt.Sprintf("There are no image files in directory %q.", dir))
		os.Exit(0)
	}

	if checkColor == nil {
		path := files[rand.Intn(len(files))]
		log.Info(path)
		err := utils.SetWallpaper(path)
		check(err)
		return
	}

	log.Info(fmt.Sprintf("Searching image closest to the color %q.", utils.Color2hex(checkColor)))

	if maxItems > 0 && maxItems < len(files) {
		rand.Shuffle(len(files), func(i, j int) {
			files[i], files[j] = files[j], files[i]
		})
		files = files[:maxItems]
	}

	imagePath := ""
	imageDistance := 500.0

	log.Debug(fmt.Sprintf("Processing %d files", len(files)))
	for i, path := range files {
		i += 1
		if i%10 == 0 {
			fmt.Print(i)
		} else {
			fmt.Print(".")
		}
		file, err := os.Open(path)
		check(err)
		defer file.Close()

		img, _, err := image.Decode(file)
		check(err)
		avgColor := average_color.AverageColor(img)
		distance := utils.GetColorDistance(&avgColor, checkColor)
		if distance < imageDistance {
			imageDistance = distance
			imagePath = path
		}
	}

	fmt.Println()
	log.Info(fmt.Sprintf("file: %q, distance: %.2f", imagePath, imageDistance))
	err := utils.SetWallpaper(imagePath)
	check(err)
}
