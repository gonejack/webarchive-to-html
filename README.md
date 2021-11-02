# webarchive-to-html

This command line converts Safari's .webarchive file to .html.

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/gonejack/webarchive-to-html)
![Build](https://github.com/gonejack/webarchive-to-html/actions/workflows/go.yml/badge.svg)
[![GitHub license](https://img.shields.io/github/license/gonejack/webarchive-to-html.svg?color=blue)](LICENSE)

### Install
```shell
> go get github.com/gonejack/webarchive-to-html
```

### Usage
```shell
> webarchive-to-html *.webarchive
```

### Flags
```
Flags:
  -h, --help        Show context-sensitive help.
  -v, --verbose     Verbose printing.
      --decorate    Append Header & Footer (not suitable for complex page) to html.
      --about       About.
```

### Tips

Chrome and Firefox would not load local images by default. 

To change their settings:

https://dev.to/dengel29/loading-local-files-in-firefox-and-chrome-m9f.

Or use Safari instead.
