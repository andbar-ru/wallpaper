This program downloads and sets random wallpaper for current screen resolution from wallhaven.cc
according to command-line arguments. The main purpose is to download wallpaper closest to the given
color.

Usage: wallhaven-wallpaper [flags] <color>

Color must be given in the hex format '#rrggbb'.

Flags:
* -h&nbsp;&nbsp;&nbsp;print help and exit
* -r&nbsp;&nbsp;&nbsp;set random wallpaper ignoring the color
* -t float&nbsp;&nbsp;&nbsp;threshold: search wallpaper so that color difference not greater than given value
* -l int &nbsp;&nbsp;&nbsp;last page. In case of random search this is the number of tryings

# Dependencies

* Go compiler
* bash
* awk
* xdpyinfo
* fbsetbg

Go packages:
* github.com/PuerkitoBio/goquery
* github.com/andbar-ru/average_color
