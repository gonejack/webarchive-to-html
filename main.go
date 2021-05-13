package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	for _, warc := range os.Args[1:] {
		err := exec.Command("textutil", "-convert", "html", warc).Run()
		if err != nil {
			log.Fatalf("syscall error: %s", err)
			return
		}

		htm := strings.TrimSuffix(warc, filepath.Ext(warc)) + ".html"
		fd, err := os.Open(htm)
		if err != nil {
			log.Fatalf("open %s error: %s", htm, err)
			return
		}

		doc, err := goquery.NewDocumentFromReader(fd)
		if err != nil {
			log.Fatalf("read %s error: %s", fd.Name(), err)
			return
		}
		_ = fd.Close()

		doc.Find("img").Each(func(i int, img *goquery.Selection) {
			src, _ := img.Attr("src")
			switch {
			case strings.HasPrefix(src, "file:///"):
				img.SetAttr("src", strings.TrimPrefix(src, "file:///"))
			}
		})

		data, err := doc.Html()
		if err != nil {
			log.Fatalf("build html error: %s", err)
			return
		}

		err = ioutil.WriteFile(htm, []byte(data), 0666)
		if err != nil {
			log.Fatalf("write %s error: %s", htm, err)
			return
		}
	}
}
