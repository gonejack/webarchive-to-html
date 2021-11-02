package main

import (
	"log"

	"github.com/gonejack/webarchive-to-html/cmd"
)

func main() {
	var c cmd.WarToHtml
	if e := c.Run(); e != nil {
		log.Fatal(e)
	}
}
