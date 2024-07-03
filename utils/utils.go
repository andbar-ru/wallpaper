package utils

import (
	"fmt"
	"image/color"
	"math"
	"os/exec"
)

func SetWallpaper(path string) error {
	cmd := exec.Command("fbsetbg", "-t", path)
	err := cmd.Run()
	return err
}

func GetColorDistance(c1, c2 *color.NRGBA) float64 {
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

func Color2hex(c *color.NRGBA) string {
	hex := fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	if c.A != 0xff {
		hex += fmt.Sprintf("%02x", c.A)
	}
	return hex
}
