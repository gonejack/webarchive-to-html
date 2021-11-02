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

func (c *WarToHtml) Run() (err error) {
	kong.Parse(&c.options,
		kong.Name("webarchive-to-html"),
		kong.Description("This command line converts .webarchive file to .html."),
		kong.UsageOnError(),
	)

	if c.About {
		fmt.Println("Visit https://github.com/gonejack/webarchive-to-html")
		return
	}

	if runtime.GOOS == "windows" {
		for _, html := range c.Wars {
			if html == "*.webarchive" {
				c.Wars = nil
				break
			}
		}
	}

	if len(c.Wars) == 0 || c.Wars[0] == "*.webarchive" {
		c.Wars, _ = filepath.Glob("*.webarchive")
	}

	return c.run()
}
func (c *WarToHtml) run() error {
	for _, war := range c.Wars {
		log.Printf("process %s", war)
		err := c.convert(war)
		if err != nil {
			return err
		}
	}
	return nil
}
func (c *WarToHtml) convert(webarchive string) (err error) {
	var warc model.WebArchive

	err = warc.From(webarchive)
	if err != nil {
		return
	}

	// extract resources
	basename := strings.TrimSuffix(filepath.Base(webarchive), filepath.Ext(webarchive))
	res, err := warc.ExtractResources(path.Join(".", fmt.Sprintf("%s_files", basename)))
	if err != nil {
		return fmt.Errorf("could not extract files: %w", err)
	}

	if c.Verbose {
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
	htmlfile := fmt.Sprintf("%s.html", basename)
	doc, err := warc.Doc(c.Decorate)
	if err != nil {
		_ = os.WriteFile(htmlfile, warc.WebMainResources.WebResourceData, 0666)
		return fmt.Errorf("parse %s error: %w", htmlfile, err)
	}
	doc.Find("img,link,script").Each(func(i int, e *goquery.Selection) { c.modRef(e, &warc, res) })

	html, err := doc.Html()
	if err != nil {
		return fmt.Errorf("build html error: %w", err)
	}

	err = os.WriteFile(htmlfile, []byte(html), 0666)
	if err != nil {
		return fmt.Errorf("write %s error: %w", htmlfile, err)
	}

	return
}
func (c *WarToHtml) modRef(e *goquery.Selection, w *model.WebArchive, res map[string]string) {
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
		local, exist = res[w.PatchRef(ref)] // try convert into absolute references
	}
	if exist {
		e.SetAttr(attr, local)
	} else {
		if c.Verbose {
			log.Printf("could not find local file of %s", ref)
		}
		e.SetAttr(attr, w.PatchRef(ref))
	}
}
