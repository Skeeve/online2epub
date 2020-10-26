package templates

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

var funcMap = template.FuncMap{
	"detag": func(txt string) string {
		return RemoveTags.ReplaceAllString(txt, ``)
	},
	"germanDate": func(format string, date time.Time) string {
		return German.Replace(date.Format(format))
	},
	"now": func(format string) string {
		return time.Now().UTC().Format(format)
	},
	"navpoint": func() func(...string) string {
		i := 0
		return func(id ...string) string {
			if len(id) < 1 {
				return "</navPoint>"
			}
			i++
			return `<navPoint id="` + strings.Replace(id[0], "-", "_", -1) + `" playOrder="` + strconv.Itoa(i) + `">`
		}
	},
}

// ContentOPF - Template used for the content.opf
var ContentOPF = newTemplate("ContentOPF", funcMap, `<?xml version="1.0" encoding="utf-8" standalone="yes"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="BookId" version="3.0">
    <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
        <dc:identifier id="BookId">{{.Ausgabe.Title}} - {{germanDate "02. Jan. 2006" .Date}}</dc:identifier>
        <dc:title>{{.Ausgabe.Title}} - {{germanDate "02. Jan. 2006" .Date}}</dc:title>
        <dc:creator id="author">ZVA Digital GmbH</dc:creator>
        <dc:publisher>Zeitungsverlag Aachen GmbH</dc:publisher>
        <dc:date>{{germanDate "2006-01-02" .Date}}</dc:date>
        <dc:language>de</dc:language>
        <meta name="cover" content="titleImage" />
        <meta property="dcterms:modified">{{now "2006-01-02T15:04:05Z" }}</meta>
    </metadata>
    <manifest>

        <item href="toc.ncx" id="ncx" media-type="application/x-dtbncx+xml" />
        <item href="title.xhtml" id="title" media-type="application/xhtml+xml" />
        <item href="index.xhtml" id="index" media-type="application/xhtml+xml" />
        {{- range .Seiten}}{{$pgidx := .Index}}

        <item href="seite_{{$pgidx}}.xhtml" id="seite_{{$pgidx}}" media-type="application/xhtml+xml" />
        {{- range .Sequence}}
        <item href="article_{{.ID}}.xhtml" id="article_{{$pgidx}}_{{.ID}}" media-type="application/xhtml+xml" />
        {{- range .Pictures}}
        {{- if .Size}}
        <item href="images/{{.ID}}.jpg" id="image_{{.ID}}" media-type="image/jpeg" />
        {{- end}}
        {{- end}}
        {{- end}}
        {{- end}}

        <item href="navigation.xhtml" id="navigation" media-type="application/xhtml+xml" properties="nav"/>
        <item href="impressum.xhtml" id="imprint" media-type="application/xhtml+xml" />
        <item href="images/title.jpg" id="titleImage" media-type="image/jpeg" />
        <item href="zva.epub.css" id="epub-stylesheet" media-type="text/css" />

    </manifest>

    <spine toc="ncx">
        <itemref idref="title" />
        <itemref idref="index" />
        {{- range .Seiten}}{{$pgidx := .Index}}

        <itemref idref="seite_{{$pgidx}}" />
        {{- range .Sequence}}
        <itemref idref="article_{{$pgidx}}_{{.ID}}" />
        {{- end}}
        {{- end}}

        <itemref idref="imprint" />
    </spine>
    <guide>
        <reference href="title.xhtml" title="Cover" type="cover" />
        <reference href="index.xhtml" title="Inhaltsverzeichnis" type="toc" />
    </guide>
</package>
`)

// Seite - Template used for pages in the newspaper
var Seite = newTemplate("Seite", funcMap, `<?xml version='1.0'?>
<!DOCTYPE html [ <!ENTITY nbsp "&#160;"> ]>
<html xmlns='http://www.w3.org/1999/xhtml'>
<head>
    <meta http-equiv='Content-Type' content='text/html; charset=UTF-8'/>
    <title>{{.Ausgabe.Title}}</title>
    <link rel='stylesheet' type='text/css' href='zva.epub.css'/>
</head>
<body>
<div class='ToC'>
    <h1 class='title'>{{html .Seite.Title}}</h1>
    {{- if .Prev.Title}}
    <a class="previous-page" href="seite_{{.Prev.Index}}.xhtml">{{html .Prev.Title}}</a>
    {{- else}}
    <a class="previous-page" href="index.xhtml">Inhalt</a>
    {{- end}}
    {{- if .Next.Title}}
    <a class="next-page" href="seite_{{.Next.Index}}.xhtml">{{html .Next.Title}}</a>
    {{- else}}
    <a class="next-page" href="impressum.xhtml">Impressum</a>
    {{- end}}
    {{- if .Seite.Sequence}}
    {{- range .Seite.Sequence}}
    <div class='ToCentry'>
        <a class='index-link' href='article_{{.ID}}.xhtml'>
            {{html .AltTitle}}
        </a>
    </div>
    {{- end}}
    <div class="source">
        <a class="external" href="{{.URL}}/#/read/{{.Ausgabe.Paper}}/{{.Ausgabe.Date}}?page={{.Seite.Index}}">
        {{germanDate "02.01.2006" .Date}} / {{.Ausgabe.Title}} / Seite {{.Seite.Number}}
        </a>
    </div>
    {{- else}}
    <div class="onlineonly">
        <p>
        Diese Seite ist leider nur
        <a class="external" href="{{.URL}}/#/read/{{.Ausgabe.Paper}}/{{.Ausgabe.Date}}?page={{.Seite.Index}}">online</a>
        oder im PDF verfügbar.
        </p>
    </div>
    {{- end}}
</div>
</body>
</html>
`)

// Index - Template used for the index page
var Index = newTemplate("Index", funcMap, `<?xml version='1.0'?>
<!DOCTYPE html [ <!ENTITY nbsp "&#160;"> ]>
<html xmlns='http://www.w3.org/1999/xhtml'>
<head>
    <meta http-equiv='Content-Type' content='text/html; charset=UTF-8'/>
    <title>{{html .Ausgabe.Title}}</title>
    <link rel='stylesheet' type='text/css' href='zva.epub.css'/>
</head>
<body>
<div class='ToC'>
    <h1 class='title'>{{html .Ausgabe.Title}}</h1>
    {{- range $index, $elt := .Ausgabe.Titles}}
    <div class='ToCentry'>
        <a class='index-link' href='seite_{{$index}}.xhtml'>
            {{html $elt}}
        </a>
    </div>
    {{- end}}
    <div class="source">
        <a class="external" href="{{.URL}}/#/read/{{.Ausgabe.Paper}}/{{.Ausgabe.Date}}">
        {{germanDate "02.01.2006" .Date}} / {{.Ausgabe.Title}}
        </a>
    </div>
</div>
</body>
</html>
`)

// ToC - Template used for the toc file
var ToC = newTemplate("ToC", funcMap, `<?xml version="1.0" encoding="UTF-8" standalone="no" ?>
{{- $navpoint := navpoint}}
<ncx version="2005-1"
    xmlns="http://www.daisy.org/z3986/2005/ncx/">
    <head>
        <meta content="{{.Ausgabe.Title}} - {{germanDate "02. Jan. 2006" .Date }}" name="dc:Title"/>
        <meta name="dtb:uid" content="{{.Ausgabe.Title}} - {{germanDate "02. Jan. 2006" .Date }}"/>
    </head>
    <docTitle>
        <text>{{.Ausgabe.Title}} - {{germanDate "02. Jan. 2006" .Date }}</text>
    </docTitle>
    <navMap>
    {{call $navpoint "startseite"}}
        <navLabel>
        <text>Startseite</text>
        </navLabel>
        <content src="title.xhtml"/>
    {{call $navpoint}}
    {{call $navpoint "Inhalt"}}
        <navLabel>
            <text>Inhalt</text>
        </navLabel>
        <content src="index.xhtml"/>
    {{call $navpoint}}
    {{- range .Seiten}}
    {{$id := printf "seite_%d" .Index}}
    {{call $navpoint $id}}
        <navLabel>
            <text>{{html .Title}}</text>
        </navLabel>
        <content src="seite_{{.Index}}.xhtml"/>
        {{- range .Sequence}}
        {{$id := printf "article_%s" .ID}}
        {{call $navpoint $id}}
            <navLabel>
                <text>{{html .AltTitle}}</text>
            </navLabel>
            <content src="{{$id}}.xhtml"/>
        {{call $navpoint}}
        {{- end}}
    {{call $navpoint}}
    {{- end}}
    {{call $navpoint "Impressum"}}
        <navLabel>
            <text>Impressum</text>
        </navLabel>
        <content src="impressum.xhtml"/>
    {{call $navpoint}}
    </navMap>
</ncx>
`)

// NAV - Template used for the nav file
var NAV = newTemplate("NAV", funcMap, `<?xml version="1.0" encoding="UTF-8" standalone="no" ?>
<!DOCTYPE html [ <!ENTITY nbsp "&#160;"> ]>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<head>
    <title>{{.Ausgabe.Title}} - {{germanDate "02. Jan. 2006" .Date }}</title>
</head>
<body>
<nav epub:type="toc">
    <h1>{{.Ausgabe.Title}} - {{germanDate "02. Jan. 2006" .Date }}</h1>
    <ol>
        <li><a href="title.xhtml">Startseite</a></li>
        <li><a href="index.xhtml">Inhalt</a></li>
        {{- range .Seiten}}
        {{$id := printf "seite_%d" .Index}}
        <li><a href="seite_{{.Index}}.xhtml">{{html .Title}}</a>
            {{- if .Sequence }}
            <ol>
                {{- range .Sequence}}
                {{$id := printf "article_%s" .ID}}
                <li><a href="{{$id}}.xhtml">{{html .AltTitle}}</a></li>
                {{- end}}
            </ol>
            {{- end}}
        </li>
        {{- end}}
        <li><a href="impressum.xhtml">Impressum</a></li>
    </ol>
</nav>
</body>
</html>
`)

// Article - Template for the newspaper articles
var Article = newTemplate("Article", funcMap, `<?xml version='1.0'?>
<!DOCTYPE html [ <!ENTITY nbsp "&#160;"> ]>
<html xmlns='http://www.w3.org/1999/xhtml'>

<head>
    <meta http-equiv='Content-Type' content='text/html; charset=UTF-8' />
    <title>{{html .A.AltTitle}}</title>
    <link rel='stylesheet' type='text/css' href='zva.epub.css' />
</head>

<body>
    <div class='article'>
        {{- if or .A.Title .A.Underline}}
        <div class='header'>
            {{if .A.Title}}<h1>{{html .A.Title}}</h1>{{end}}
            {{if .A.Underline}}{{.A.Underline}}{{end}}
        </div>
        {{- end}}
        {{- if .A.Pictures}}
            {{- range .A.Pictures}}
        <div class="image">
            {{- if .Size}}
            <img src="images/{{.ID}}.jpg" alt="ID={{.ID}}"/>
            {{- else}}
            <p class="imgerr">Dieses Bild konnte nicht geladen werden</p>
            {{- end}}
            {{- if .Description}}
            <p class="imgdescription">{{.Description}}</p>
            {{- end}}
        </div>
            {{- end}}
        {{- end}}
        {{- if .A.Author}}
        <div class='author'>
            {{.A.Author}}
        </div>
        {{- end}}
        {{- if .A.Text}}
        <div class='content'>
            {{.A.Text}}
        </div>
        {{- end}}
        {{- if ne .A.ID "Impressum"}}
        <div class="source">
            <a class="external" href="{{.URL}}/#/read/{{.A.Paper.Paper}}/{{.A.Paper.Date}}?page={{.A.Paper.Page.Index}}&amp;article={{.A.ID}}">
            {{germanDate "02.01.2006" .Date}} / {{.A.Paper.Title}} / Seite {{.A.Paper.Page.Number}} / {{html .A.Paper.Page.Title}}
            </a>
        </div>
        {{- end}}
    </div>
</body>

</html>
`)

// Imprint - The Impressum's template
var Imprint = newTemplate("Impressum", funcMap, `<?xml version='1.0'?>
<!DOCTYPE html [ <!ENTITY nbsp "&#160;"> ]>
<html xmlns='http://www.w3.org/1999/xhtml'>

<head>
    <meta http-equiv='Content-Type' content='text/html; charset=UTF-8' />
    <title>Impresum</title>
    <link rel='stylesheet' type='text/css' href='zva.epub.css' />
</head>

<body>
    <div class='article'>
        <div class='header'>
            <h1>Impressum</h1>
        </div>
        <div class='content'>
            {{.Text}}
        </div>
    </div>
</body>

</html>
`)

// Bildnamen - Maps an unnamed picture's size to a specific name
var Bildnamen = strings.NewReplacer(
	"Bild 296 × 591", "Festgeld",
	"Bild 1024 × 460", "DAX",
	"Bild 1024 × 360", "Rätsel Ecke",
	"Bild 1024 × 361", "Rätsel Ecke",
	"Bild 1024 × 388", "Popel",
	"Bild 1024 × 788", "Wetter",
	"Bild 361 × 818", "Kinder-Sudoku",
	"Bild 1024 × 411", "Finde die Unterschiede",
)

// German - Replaces english names and abbreviations in dates
// with the german translations
var German = strings.NewReplacer(
	"January", "Januar",
	"February", "Februar",
	"March", "März", "Mar", "Mär",
	"April", "April",
	"May", "Mai", "May", "Mai",
	"June", "Juni",
	"July", "Juli",
	"August", "August",
	"September", "September",
	"October", "Oktober", "Oct", "Okt",
	"November", "November",
	"December", "Dezember", "Dec", "Dez",
	"Monday", "Montag",
	"Tuesday", "Dienstag", "Tue", "Die",
	"Wednesday", "Mittwoch", "Wed", "Mit",
	"Thursday", "Donnerstag", "Thu", "Don",
	"Friday", "Freitag", "Fri", "Fre",
	"Saturday", "Samstag", "Sat", "Sam",
	"Sunday", "Sontag", "Sun", "Son",
)

const (
	// TitlePage - The fixed content of the newspaper's title page
	// The only thing changing on that page is the content of
	// the title image. Its name is always the same
	TitlePage = `<?xml version='1.0'?>
<!DOCTYPE html [ <!ENTITY nbsp "&#160;"> ]>
    <html xmlns='http://www.w3.org/1999/xhtml'>
    <head>
    <meta http-equiv='Content-Type' content='text/html; charset=UTF-8' />
    <title>Titelseite</title>
    <link rel='stylesheet' type='text/css' href='zva.epub.css' />
    </head>
    <body>
    <div id='content'>
        <img src='images/title.jpg' id='teaser-image' alt='Titelbild' />
    </div>
    </body>
</html>
`
	// ContainerXML - The fixed content of the container.xml file
	ContainerXML = `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
    <rootfiles>
        <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
    </rootfiles>
</container>
`
	// ZvaCSS - the fixed content of the newspaper's CSS
	ZvaCSS = `h1 {
    font-size: 2.17rem;
    margin-bottom: .42rem;
}

h3 {
    font-family: Lato,Calibri,Roboto,Arial,sans-serif;
    font-weight: 900;
    font-size: 0.84rem;
    margin-bottom: 0.84rem;
}

.ToC,
.article {
    font-family: "Times New Roman",Times,serif;
    line-height: 1;
}

.ToC a,
.article a {
    text-decoration: none;
    color: #278bcf;
}

.ToC .header,
.article .header {
    display: block;
}
.article .header p {
    font-size: 0.83rem;
    margin-bottom: 1rem;
}
.article .author {
    color: #96969b;
    font-size: 0.67rem;
    font-family: Roboto,Helvetica,sans-serif;
    line-height: 1.2;
    text-transform: uppercase;
    display: block;
    border-bottom: 1px solid #96969b;
}

.article .IR_AZAN-Infobox_Balken {
    background-color: #0085c7;
    padding-left: 5px;
    text-transform: uppercase;
    color: #fff;
    font-weight: 700;
}

.article .fotocredit {
    text-transform: uppercase;
    font-size: 0.83rem;
}
.article .content {
    font-size: 1rem;
    line-height: 1.3;
}

.article .content .quote {
    text-align:center;
    padding:.5rem 3.33rem;
    margin-bottom:1.08rem;
    font-family:Lato,Calibri,Roboto,Arial,sans-serif;
    font-weight:900;
    font-size:1rem;
}

.article .content .quote p > i {
    font-style: normal;
    font-size: 0.92rem;
    font-weight: 700;
}

b, strong {
    font-weight: 400;
}

.article .content p > b {
    font-size: .67em;
    font-weight: 900;
    font-family: Lato,Calibri,Roboto,Arial,sans-serif;
}

.article .content .quote p:empty {
    display:none
}
    
.article .content .box {
    padding: 0.83rem 1.25rem;
    border-top: 1px solid #efeff5;
}

.article .content .box p {
    font-family: Lato,Calibri,Roboto,Arial,sans-serif;
    font-size: 16px;
    line-height: 1.5;
}


.article .content p {
    margin-bottom: 1.04rem
}

.article .ortsmarke {
    font-weight: 900;
}
.article .ortsmarke::before {
    content: "("
}
.article .ortsmarke::after {
    content: ")"
}

.ToC .source,
.article .source {
    font-family: sans-serif;
    font-size: 0.67rem;
    width: 100%;
    text-align: right;
}

a.external::after {
    content: "\202f\279a"
}

.article .image {
    margin: 0;
}

.article .image img {
    max-height: 50vh;
    display: block;
    margin: auto;
}

.article .image .imgerr {
    color: red;
    font-size: 0.67rem;
    font-style: italic;
    text-align: center;
} 

.article .image .imgerr::before {
    color: red;
    font-size: 1rem;
    font-style: normal;
    content: "\26a0\202f"
} 

.article .image .imgdescription {
    display: block;
    text-align: center;
}

.ToC a.previous-page {
    float: left;
}
.ToC a.previous-page::before {
    content: "\25c0\202f";
}

.ToC a.next-page {
    float: right;
}

.ToC a.next-page::after {
    content: "\202f\25b6";
}

.ToC .onlineonly {
    clear: both;
}

.ToC .ToCentry {
    width: 100%;
    border-bottom: 1px solid #ddd;
    padding: 0.25rem;
    clear: both;
}

.ToC .ToCentry:first-of-type {
    border-top: 1px solid #ddd;
    margin-top: 1-08rem;
}

.ToC .source {
    margin-top: 1rem;
    clear: both;
    }
`
)

// The following regular expressions are used
// to create a "headline" for an untitled article.

// KillFirstTag - remove the first tag element of an article
var KillFirstTag = regexp.MustCompile(`^<(\w+)\b[^>]*>`)

// CutOffParagraphs - remove everything after an article's first paragraph
var CutOffParagraphs = regexp.MustCompile(`</p>.*$`)

// RemoveLocationMark - Removes the location mark
var RemoveLocationMark = regexp.MustCompile(`^<b\s+class="ortsmarke">.*?</b>\s*`)

// RemoveTags - removes all tags
var RemoveTags = regexp.MustCompile(`<[^>]+>`)

// Shorten - shorten to at most 40 characters
var Shorten = regexp.MustCompile(`^(.{0,40}\S*).*`)

// NoLinkTarget - renove all target attributes from links
var NoLinkTarget = regexp.MustCompile(`(<a\b[^>]*)\starget=".*?"`)

// MTbr - replace all br with proper empty tags
var MTbr = regexp.MustCompile(`<(br)\s*>`)

func newTemplate(name string, funcMap template.FuncMap, tpl string) *template.Template {
	result, err := template.New(name).Funcs(funcMap).Parse(tpl)
	if err != nil {
		fmt.Println(err)
		os.Exit(5)
	}
	return result
}
