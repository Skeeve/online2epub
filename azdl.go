package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"hradek.net/azdl/templates"
)

const baseURL = "https://epaper.zeitungsverlag-aachen.de/2.0"

var standardHeaders = map[string]string{
	"User-Agent":   "Stephan Hradeks AZAN Epub Builder v0.0",
	"Content-Type": "application/json;charset=utf-8",
	"Accept":       "application/json",
}

type client struct {
	C            *http.Client
	Header       http.Header
	Impressum    string
	Ausgabe      string
	BaseURL      string
	NewspaperURL string
	Ed2Name      map[string]string
	Name2Ed      map[string]string
}

type azanlogin struct {
	Login string `json:"login"`
	Pass  string `json:"password"`
}

type azanauthorization struct {
	Authorization string `json:"authorizationHeader"`
	Error         string `json:"error"`
}

type ausgabe struct {
	Paper        string   `json:"paper"`         // az-d
	Title        string   `json:"title"`         // Dürener Zeitung
	Date         int      `json:"date"`          // 20060102
	Brand        string   `json:"brand"`         // az
	Pages        int      `json:"numberOfPages"` // 14
	Titles       []string `json:"pageTitles"`    // ["TITELSEITE", "POLITIK"…]
	Subscription bool     `json:"subscription"`  // true
	Bought       bool     `json:"bought"`        // false
	Version      int      `json:"version"`       // 1598303355
}

type seite struct {
	ID       string    `json:"id"`       //  "20200820-47208377",
	Title    string    `json:"title"`    //  "DIE SEITE DREI",
	Number   int       `json:"number"`   //  3,
	Index    int       `json:"index"`    //  2,
	Width    int       `json:"width"`    //  351,
	Height   int       `json:"height"`   //  506,
	Elements []element `json:"elements"` //
	Free     bool      `json:"free"`     //  false
	Sequence []element
}

type element struct {
	ID        string `json:"id"`        //  "88937601",
	XStart    int    `json:"xStart"`    //  12,
	XEnd      int    `json:"xEnd"`      //  283,
	YStart    int    `json:"yStart"`    //  45,
	YEnd      int    `json:"yEnd"`      //  492,
	Area      int    `json:"area"`      //  121137,
	Width     int    `json:"width"`     //  620,
	Height    int    `json:"height"`    //  1024,
	Type      string `json:"type"`      //  "article",
	Title     string `json:"title"`     //  "Corona-Hotspot Innenraum",
	Author    string `json:"author"`    //  "",
	Underline string `json:"underline"` //  "An der frischen Luft …"
	Headline  string `json:"headline"`  //  "",
	Location  string `json:"location"`  //  ""
	AltTitle  string
	Pictures  []picture
}

type picture struct {
	ID          string `json:"id"`          //  "2094290259_e7c39b54a0.irprodgera_i14u8q",
	XStart      int    `json:"xStart"`      //  12,
	XEnd        int    `json:"xEnd"`        //  62,
	YStart      int    `json:"yStart"`      //  57,
	YEnd        int    `json:"yEnd"`        //  93,
	Area        int    `json:"area"`        //  1800,
	Width       int    `json:"width"`       //  600,
	Height      int    `json:"height"`      //  429,
	Type        string `json:"type"`        //  "picture",
	Description string `json:"description"` //  null
	Size        int64
}

type page struct {
	ID     string `json:"id"`     // "20200821-47213513",
	Index  int    `json:"index"`  // 11,
	Number int    `json:"number"` // 12,
	Title  string `json:"title"`  // "LOKALES"
}

type link struct {
	ID    string `json:"id"`
	Paper paper  `json:"paper"`
}

type paper struct {
	Paper string `json:"paper"` // "az-d",
	Date  string `json:"date"`  // "20200821",
	Title string `json:"title"` // "D\u00fcrener Zeitung",
	Page  page   `json:"page"`
}

type article struct {
	ID         string    `json:"id"`        //  "88973299",
	XStart     int       `json:"xStart"`    //  12,
	XEnd       int       `json:"xEnd"`      //  109,
	YStart     int       `json:"yStart"`    //  57,
	YEnd       int       `json:"yEnd"`      //  93,
	Area       int       `json:"area"`      //  3492,
	Width      int       `json:"width"`     //  570,
	Height     int       `json:"height"`    //  209,
	Type       string    `json:"type"`      //  "article",
	Title      string    `json:"title"`     //  "",
	Author     string    `json:"author"`    //  "",
	Underline  string    `json:"underline"` //  "",
	Headline   string    `json:"headline"`  //  "",
	Location   string    `json:"location"`  //  "",
	Pictures   []picture `json:"pictures"`
	Paper      paper     `json:"paper"`
	Text       string    `json:"text"`       //  "<p>Joe Biden<\/p><p>Der Mann, der Donald Trump<br \/>als US-Pr\u00e4sident abl\u00f6sen will<\/p><p>Die Seite Drei<\/p>",
	Sociallink string    `json:"sociallink"` //  "https:\/\/epaper.zeitungsverlag-aachen.de\/2.0\/article\/327f34db08",
	Print      string    `json:"print"`      //  "https:\/\/epaper.zeitungsverlag-aachen.de\/2.0\/article\/327f34db08",
	Wordcount  int       `json:"wordcount"`  //  13
	Prev       link      `json:"prev"`
	Next       link      `json:"next"`
	AltTitle   string
}

type pgInfo struct {
	Title string
	Index int
}

func main() {

	client := ePaperClient()

	i := 1
	ausgabe := ""
	if len(os.Args) > i {
		mtch, _ := regexp.MatchString(`^(?:latest|\d{8})$`, os.Args[i])
		if !mtch {
			ausgabe = os.Args[i]
			i++
		}
	}
	if ausgabe == "" {
		var ok bool
		ausgabe, ok = os.LookupEnv("AZAN_AUSGABE")
		if !ok {
			log.Fatal("Umgebungsvariable AZAN_AUSGABE fehlt")
		}
	} else if ausgabe == "-?" {
		keys := make([]string, 0, len(client.Name2Ed))
		for k := range client.Name2Ed {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("%-6s: %s\n", client.Name2Ed[k], k)
		}
		os.Exit(0)
	}
	client.ePaperLogin(ausgabe)

	if len(os.Args) < 1+i {
		client.createAzanEpub("latest")
		os.Exit(0)
	}
	for i < len(os.Args) {
		client.createAzanEpub(os.Args[i])
		i++
	}
	os.Exit(0)
}

func (c *client) createAzanEpub(wantedDate string) {

	// Hole die Basisdatei der gewünschten Ausgabe
	zeitung := new(ausgabe)
	c.getJSON(wantedDate, zeitung)

	if !zeitung.Subscription && !zeitung.Bought {
		log.Fatal(fmt.Sprintf("Die %s wurde weder abonniert noch gekauft", zeitung.Title))
	}

	// Das Datum ist als String in der Ausgabe hinterlegt
	strdate := strconv.Itoa(zeitung.Date)
	date, _ := time.Parse("20060102", strdate)

	// Erstelle eine Datei für das ePub
	filename := c.Ausgabe + date.Format("-2006-01-02") + ".epub"
	epubFile, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer epubFile.Close()

	azanEpub := zip.NewWriter(epubFile)

	// ToDo: Eventuell müssen Verzeichnisse erstellt werden...
	// os.MkdirAll("epub/META-INF", 0755)
	// os.MkdirAll("epub/OEBPS/images", 0755)

	// Füge einige Standard Dateien zum ePub hinzu archive.
	zipString(azanEpub, "mimetype", "application/epub+zip")
	c.saveFromURL(azanEpub, strdate+"/0/big", "OEBPS/images/title.jpg")
	zipString(azanEpub, "OEBPS/title.xhtml", templates.TitlePage)
	zipString(azanEpub, "OEBPS/zva.epub.css", templates.ZvaCSS)
	zipString(azanEpub, "META-INF/container.xml", templates.ContainerXML)
	writeTemplate(azanEpub, "OEBPS/impressum.xhtml", templates.Imprint, struct{ Text string }{c.Impressum})
	writeTemplate(azanEpub, "OEBPS/index.xhtml", templates.Index, struct {
		URL     string
		Ausgabe *ausgabe
		Date    time.Time
	}{
		c.BaseURL,
		zeitung,
		date,
	})

	// Array für die Seiten
	seiten := make([]*seite, zeitung.Pages)
	// map für die Artikel
	alleArtikel := make(map[string]*article)

	// Durch alle Seiten iterieren
	for i := 0; i < zeitung.Pages; i++ {
		fmt.Print(" ", i, "\r")

		// relative URL der Seite
		seitenURL := strdate + "/" + strconv.Itoa(i)

		// Seite laden
		dieseSeite := new(seite)
		c.getJSON(seitenURL, dieseSeite)
		seiten[i] = dieseSeite

		// Anhand der Verlinkung wird ermittelt,
		// Welcher Artikel auf derSeite der
		// erste sein soll
		ersterArtikel := -1

		// Für die Ermittlung der Reihenfolge müssen wir
		// von der Artikel-ID auf ihren Index
		id2idx := make(map[string]int)
		// und vom Index auf die Artikel-ID des Folgeartikels
		// schließen können
		idx2next := make([]string, len(dieseSeite.Elements))

		// iteriere durch die Seitenelemente
		for idx, element := range dieseSeite.Elements {
			// Wir laden nur Titel, Keine Werbung, keine Bilder
			switch element.Type {
			case "article":
				// Hole den Artikel
				artikel := new(article)
				c.getJSON(seitenURL+"/"+element.ID, artikel)

				// Verknüpfungen
				alleArtikel[artikel.ID] = artikel
				id2idx[artikel.ID] = idx
				idx2next[idx] = artikel.Next.ID

				// Alternativtitel erstellen aus
				// dem Inhalt des Artikels
				altTitle := cheapExerpt(artikel)
				clean(artikel)
				artikel.AltTitle = altTitle
				dieseSeite.Elements[idx].AltTitle = altTitle
				dieseSeite.Elements[idx].Pictures = artikel.Pictures

				// Bilder holen
				for idx, picture := range artikel.Pictures {
					size := c.saveFromURL(azanEpub, seitenURL+"/"+picture.ID+"/jpg", "OEBPS/images/"+picture.ID+".jpg")
					artikel.Pictures[idx].Size = size
					if size < 1 {
						fmt.Println("Fehlendes Bild Seite ", dieseSeite.Number, " ", altTitle)
					}
				}

				// xhtml Datei für den Artikel erstellen
				date, _ := time.Parse("20060102", artikel.Paper.Date)
				writeTemplate(azanEpub, "OEBPS/article_"+artikel.ID+".xhtml", templates.Article, struct {
					URL  string
					A    *article
					Date time.Time
				}{
					c.BaseURL,
					artikel,
					date,
				})

				// Prüfe, ob es sich um den ersten Artikel handelt
				if ersterArtikel < 0 && (artikel.Prev.ID == "" || artikel.Prev.Paper.Page.Index < artikel.Paper.Page.Index) {
					ersterArtikel = idx
				}

			default:
				// Alles ausser article wird ignoriert.
			}
		}
		// Reihenfolge der Artikel auf der Seite ermitteln
		dieseSeite.Sequence = make([]element, len(id2idx))
		art := ersterArtikel
		for i := 0; art >= 0; i++ {
			dieseSeite.Sequence[i] = dieseSeite.Elements[art]
			nextID := idx2next[art]
			if nextID == "" {
				break
			}
			var ok bool
			if art, ok = id2idx[nextID]; !ok {
				break
			}
		}

		// Vorgänger und Nachfolger für
		// die Inhaltsangaben der Seiten
		var nextPage pgInfo
		if i+1 < zeitung.Pages {
			nextPage.Index = i + 1
			nextPage.Title = zeitung.Titles[i+1]
		}
		var prevPage pgInfo
		if i > 0 {
			prevPage.Index = i - 1
			prevPage.Title = zeitung.Titles[i-1]
		}

		// Inhaltsangabe der Seite erstellen
		writeTemplate(azanEpub, "OEBPS/seite_"+strconv.Itoa(dieseSeite.Index)+".xhtml", templates.Seite, struct {
			URL     string
			Ausgabe *ausgabe
			Seite   *seite
			Date    time.Time
			Prev    pgInfo
			Next    pgInfo
		}{
			c.BaseURL,
			zeitung,
			dieseSeite,
			date,
			prevPage,
			nextPage,
		})
	}

	// Daten für Table Of Content etc.
	data := struct {
		URL     string
		Ausgabe *ausgabe
		Seiten  []*seite
		Date    time.Time
	}{
		c.BaseURL,
		zeitung,
		seiten,
		date,
	}

	// ePub Steuerdateien erstellen
	writeTemplate(azanEpub, "OEBPS/toc.ncx", templates.ToC, data)
	writeTemplate(azanEpub, "OEBPS/content.opf", templates.ContentOPF, data)
	writeTemplate(azanEpub, "OEBPS/navigation.xhtml", templates.NAV, data)

	// Make sure to check the error on Close.
	err = azanEpub.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func newTemplate(name string, funcMap template.FuncMap, tpl string) *template.Template {
	result, err := template.New(name).Funcs(funcMap).Parse(tpl)
	if err != nil {
		fmt.Println(err)
		os.Exit(5)
	}
	return result
}

func writeTemplate(zipWriter *zip.Writer, filename string, tpl *template.Template, data interface{}) {
	f, err := zipWriter.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	err = tpl.Execute(f, data)
	if err != nil {
		log.Fatal(err)
	}
}

func (c *client) saveFromURL(zipWriter *zip.Writer, relativeURL, filename string) int64 {
	request, _ := http.NewRequest("GET", c.NewspaperURL+"/"+relativeURL, nil)
	request.Header = c.Header.Clone()
	response, err := c.C.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	size := response.ContentLength
	if 0 == size {
		return 0
	}
	f, err := zipWriter.Create(filename)
	if err != nil {
		log.Fatal(err)
	}

	_, err = io.Copy(f, response.Body)
	if err != nil {
		log.Fatal(err)
	}
	return size
}

func (c *client) getInfos(impressumURL string) {
	request, err := http.NewRequest("GET", c.BaseURL+impressumURL, nil)
	request.Header = c.Header.Clone()
	response, err := c.C.Do(request)
	if err != nil {
		log.Fatal(err)
		os.Exit(3)
	}
	defer response.Body.Close()

	// den response body als string holen
	dataInBytes, err := ioutil.ReadAll(response.Body)
	pageContent := string(dataInBytes)

	// Impressum suchen
	indicator := "<h1>Impressum</h1>"
	impressumPos := strings.Index(pageContent, indicator)
	if impressumPos == -1 {
		fmt.Printf("Kein Impressum in %s gefunden.\n", impressumURL)
	} else {
		// Das Ende des Impressum suchen
		impressumPos += len(indicator)
		impressumEnd := strings.Index(pageContent[impressumPos:], "')")
		if impressumEnd == -1 {
			impressumEnd = len(pageContent)
		} else {
			impressumEnd += impressumPos
		}

		// br und Zeilenenden erstzen und das als Ergebnis liefern
		c.Impressum = strings.Replace(strings.Replace(
			pageContent[impressumPos:impressumEnd],
			"<br>", "<br />", -1),
			"\\n", "\n", -1)
	}
	// Editionen suchen
	editions := regexp.MustCompile(`paper:"([^"]+)",title:"([^"]+)",`).FindAllStringSubmatch(pageContent, -1)
	for _, edition := range editions {
		c.Ed2Name[edition[1]] = edition[2]
		c.Name2Ed[edition[2]] = edition[1]
	}
}

// Artikel "reinigen"
func clean(artikel *article) {
	artikel.Text =
		templates.Ortsmarke.ReplaceAllString(
			templates.MTbr.ReplaceAllString(
				templates.NoLinkTarget.ReplaceAllString(
					artikel.Text,
					`$1`),
				`$1`),
			`$1$3$2`)
}

// Erstellen eines Alternativtitels
func cheapExerpt(artikel *article) string {
	// Den vorhandenen Titel nehmen
	if artikel.Title != "" {
		return artikel.Title
	}
	// Sonst, wenn kein Artikeltext vorhanden
	txt := artikel.Text
	if txt == "" {
		// Wenn kein Bild vorhandeen is
		if len(artikel.Pictures) < 1 {
			return "Leerer Artikel"
		}
		// Und keine Bildbeschreibung fürs erste Bild
		txt = artikel.Pictures[0].Description
		if txt == "" {
			// Dann generiere einen Bildnamen aus den Dimensionen des Bildes
			// Bekannte Dimensionen (DAX, Wetter, Festgeld...)
			// werden durch feste Namen ersetzt. Siehe templates.go
			bildname := templates.Bildnamen.Replace(
				fmt.Sprintf("Bild %d × %d", artikel.Width, artikel.Height))
			fmt.Println(artikel.Pictures[0].ID, " ", bildname)
			return bildname
		}
	}
	// Aufbereiten des Textes
	// - Erstes Tag entfernen
	// - Ab dem ersten </p> alles abschneiden
	// - "Locationmark" entfernen
	// - Alle Tags löschen
	txt = templates.RemoveTags.ReplaceAllString(
		templates.RemoveLocationMark.ReplaceAllString(
			templates.CutOffParagraphs.ReplaceAllString(
				templates.KillFirstTag.ReplaceAllString(txt,
					``),
				``),
			``),
		``)
	// Texte über 40 Zeichen länge kürzen
	if len(txt) > 40 {
		txt = templates.Shorten.ReplaceAllString(txt, `$1…`)
	}
	return txt
}

func ePaperClient() *client {
	// Erstelle HTTP client
	myclient := client{
		C: &http.Client{
			Timeout: 30 * time.Second,
		},
		Header:  http.Header{},
		BaseURL: baseURL,
		Ed2Name: make(map[string]string),
		Name2Ed: make(map[string]string),
	}
	for k, v := range standardHeaders {
		myclient.Header.Set(k, v)
	}
	myclient.BaseURL = baseURL

	// Impressum und Editionen laden
	myclient.getInfos("/js/app-b4b5468874.js")

	return &myclient
}

func (c *client) ePaperLogin(azanAusgabe string) {
	edTitel := c.Ed2Name[azanAusgabe]
	edition := c.Name2Ed[azanAusgabe]
	if edTitel != "" {
		edition = azanAusgabe
	} else if edition != "" {
		edTitel = azanAusgabe
	} else {
		log.Fatal(fmt.Sprintf("Es gibt keine Ausgabe %s der Aachener Zeitung.\n", azanAusgabe))
	}
	apiURL := c.BaseURL + "/api/"
	c.NewspaperURL = apiURL + edition
	c.Ausgabe = edition
	// Die credentials aus dem Environment holen
	body := &azanlogin{
		Login: os.Getenv("AZAN_USER"),
		Pass:  os.Getenv("AZAN_PASS"),
	}
	// and prepare for login
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(body)

	// Log in
	request, err := http.NewRequest("POST", apiURL+"user/login", buf)
	if err != nil {
		log.Fatal(err)
	}

	auth := new(azanauthorization)
	// Make login request
	c.fetchJSON(request, auth)
	if auth.Error != "" {
		log.Fatal(auth.Error)
	}

	// Set authorization
	c.Header.Set("Authorization", auth.Authorization)
}

func (c *client) getJSON(relativeURL string, target interface{}) {
	request, _ := http.NewRequest("GET", c.NewspaperURL+"/"+relativeURL, nil)
	c.fetchJSON(request, target)
}
func (c *client) fetchJSON(request *http.Request, target interface{}) {
	request.Header = c.Header.Clone()
	response, err := c.C.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	// parse the page data
	json.NewDecoder(response.Body).Decode(target)
}

func zipString(zipWriter *zip.Writer, filename string, content string) {

	var f io.Writer
	var err error
	if filename == "mimetype" {
		f, err = zipWriter.CreateHeader(&zip.FileHeader{
			Name:   filename,
			Method: zip.Store,
		})
	} else {
		f, err = zipWriter.Create(filename)
	}
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.Write([]byte(content))
	if err != nil {
		log.Fatal(err)
	}
}
