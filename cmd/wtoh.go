package cmd

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/alecthomas/kong"

	"github.com/gonejack/webarchive-to-html/model"
)

type options struct {
	Verbose  bool     `short:"v" help:"Verbose printing."`
	Decorate bool     `help:"Append Header & Footer (not suitable for complex page) to html."`
	About    bool     `help:"About."`
	Wars     []string `arg:"" optional:""`
}

type WarToHtml struct {
	options
}

func (w *WarToHtml) Run() (err error) {
	kong.Parse(&w.options,
		kong.Name("webarchive-to-html"),
		kong.Description("This command line converts .webarchive file to .html."),
		kong.UsageOnError(),
	)

	if w.About {
		fmt.Println("Visit https://github.com/gonejack/webarchive-to-html")
		return
	}

	if runtime.GOOS == "windows" {
		for _, html := range w.Wars {
			if html == "*.webarchive" {
				w.Wars = nil
				break
			}
		}
	}

	if len(w.Wars) == 0 || w.Wars[0] == "*.webarchive" {
		w.Wars, _ = filepath.Glob("*.webarchive")
	}

	return w.run()
}
func (w *WarToHtml) run() error {
	for _, war := range w.Wars {
		log.Printf("process %s", war)
		err := w.convert(war)
		if err != nil {
			return err
		}
	}
	return nil
}
func (w *WarToHtml) convert(webarchive string) (err error) {
	var warc model.WebArchive

	err = warc.From(webarchive)
	if err != nil {
		return
	}

	// extract resources
	name := strings.TrimSuffix(filepath.Base(webarchive), filepath.Ext(webarchive))
	html := fmt.Sprintf("%s.html", name)
	res, err := warc.ExtractResources(path.Join(".", fmt.Sprintf("%s_files", name)))
	if err != nil {
		return fmt.Errorf("could not extract files: %w", err)
	}

	if w.Verbose {
		for ref, local := range res {
			if strings.HasPrefix(ref, "data:") {
				tmp := []rune(ref)
				if len(tmp) > 70 {
					ref = string(tmp[:70]) + "..."
				}
			}
			log.Printf("save %s as %s", ref, local)
		}
	}

	// parse html
	doc, err := warc.Doc(w.Decorate)
	if err != nil {
		_ = os.WriteFile(html, warc.WebMainResources.WebResourceData, 0666)
		return fmt.Errorf("parse %s error: %w", html, err)
	}
	doc.Find("img,link,script").Each(func(i int, e *goquery.Selection) { w.modifyRef(e, &warc, res) })

	patched, err := doc.Html()
	if err != nil {
		return fmt.Errorf("build html error: %w", err)
	}

	err = os.WriteFile(html, []byte(patched), 0666)
	if err != nil {
		return fmt.Errorf("write %s error: %w", html, err)
	}

	return
}
func (w *WarToHtml) modifyRef(e *goquery.Selection, warc *model.WebArchive, res map[string]string) {
	attr := "src"
	switch e.Get(0).Data {
	case "link":
		rel, _ := e.Attr("rel")
		if rel == "canonical" {
			return
		}
		attr = "href"
	}

	ref, _ := e.Attr(attr)
	if ref == "" {
		return
	}

	local, exist := res[ref]
	if !exist {
		// try convert into absolute references
		local, exist = res[warc.PatchRef(ref)]
	}
	if !exist {
		if w.Verbose {
			log.Printf("could not find local file of %s", ref)
		}
		return
	}

	e.SetAttr(attr, local)
}
