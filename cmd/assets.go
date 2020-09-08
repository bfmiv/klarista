package cmd

import (
	"github.com/gobuffalo/packr/v2"
)

var assets *packr.Box

func init() {
	assets = packr.New("assets", "../assets")
}
