package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"howett.net/plist"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		args, _ = filepath.Glob("*.webarchive")
	}

	for _, w := range args {
		log.Printf("process %s", w)
		warc, err := filepath.Abs(w) // convert to absolute path
		if err != nil {
			log.Fatalf("resolve abs path of %s error: %s", w, err)
			return
		}
		err = convert2(warc)
		if err != nil {
			log.Fatalf("process %s error: %s", w, err)
			return
		}
	}
}

func convert(warc string) (err error) {
	// make resource dir
	name := strings.TrimSuffix(filepath.Base(warc), filepath.Ext(warc))
	resd := filepath.Join(".", fmt.Sprintf("%s_files", name))

	err = os.MkdirAll(resd, 0766)
	if err != nil {
		return fmt.Errorf("mkdir %s error: %w", resd, err)
	}

	// link webarchive file to resd/temp.warc
	warcLink := filepath.Join(resd, "temp.warc")
	err = os.Symlink(warc, warcLink)
	if err != nil {
		return fmt.Errorf("link %s => %s error: %w", warc, warcLink, err)
	}
	defer os.Remove(warcLink)

	// run textutil on resd/temp.warc
	err = exec.Command("textutil", "-convert", "html", warcLink).Run()
	if err != nil {
		return fmt.Errorf("syscall error: %w", err)
	}

	// modify content of resd/temp.html
	html := filepath.Join(resd, "temp.html")
	fd, err := os.Open(html)
	if err != nil {
		return fmt.Errorf("open %s error: %w", html, err)
	}

	doc, err := goquery.NewDocumentFromReader(fd)
	if err != nil {
		return fmt.Errorf("read %s error: %w", fd.Name(), err)
	}
	_ = fd.Close()

	doc.Find("img").Each(func(i int, img *goquery.Selection) {
		src, _ := img.Attr("src")
		switch {
		case strings.HasPrefix(src, "file:///"):
			src = strings.TrimPrefix(src, "file:///")
			src = path.Join(resd, src)
			img.SetAttr("src", src)
		}
	})

	data, err := doc.Html()
	if err != nil {
		return fmt.Errorf("build html error: %w", err)
	}

	err = ioutil.WriteFile(html, []byte(data), 0666)
	if err != nil {
		return fmt.Errorf("write %s error: %w", html, err)
	}

	// rename resd/temp.html to ./{name}.html
	err = os.Rename(html, filepath.Join(".", fmt.Sprintf("%s.html", name)))
	if err != nil {
		return fmt.Errorf("move %s error: %w", html, err)
	}

	return
}

func convert2(warc string) (err error) {
	fd, err := os.Open(warc)
	if err != nil {
		return
	}
	defer fd.Close()

	var w WebArchive
	err = plist.NewDecoder(fd).Decode(&w)
	if err != nil {
		return
	}

	// make resource dir
	name := strings.TrimSuffix(filepath.Base(warc), filepath.Ext(warc))
	html := fmt.Sprintf("%s.html", name)
	resd := filepath.Join(".", fmt.Sprintf("%s_files", name))
	err = os.MkdirAll(resd, 0766)
	if err != nil {
		return fmt.Errorf("mkdir %s error: %w", resd, err)
	}

	// get html
	doc, err := w.Doc()
	if err != nil {
		_ = ioutil.WriteFile(html, w.WebMainResources.WebResourceData, 0666)
		return fmt.Errorf("parse %s error: %w", html, err)
	}

	// process html
	doc.Find("img,link,script").Each(func(i int, e *goquery.Selection) {
		var attr string
		switch e.Get(0).Data {
		case "img":
			attr = "src"
		case "link":
			rel, _ := e.Attr("rel")
			if rel == "canonical" {
				return
			}
			attr = "href"
		case "script":
			attr = "src"
		}

		src, _ := e.Attr(attr)
		switch {
		case src == "":
			return
		case strings.HasPrefix(src, "data:"):
			return
		}

		// convert into absolute references
		if !strings.HasPrefix(src, "http") {
			src = w.patchRef(src)
			e.SetAttr(attr, src)
		}

		local := path.Join(resd, md5str(src)+path.Ext(src))
		fd, err := os.Open(local)
		switch {
		case err == nil: // file exist
			fd.Close()
		case errors.Is(err, fs.ErrNotExist):
			res, exist := w.FindResource(src)
			if !exist {
				log.Printf("resource %s not exist", src)
				return
			}
			err = ioutil.WriteFile(local, res.WebResourceData, 0666)
			if err != nil {
				log.Fatalf("cannot write %s: %s", local, err)
				return
			}
		default:
			log.Printf("cannot open %s: %s", local, err)
			return
		}

		e.SetAttr(attr, local)
	})

	data, err := doc.Html()
	if err != nil {
		return fmt.Errorf("build html error: %w", err)
	}

	err = ioutil.WriteFile(html, []byte(data), 0666)
	if err != nil {
		return fmt.Errorf("write %s error: %w", html, err)
	}

	return
}

func md5str(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}
