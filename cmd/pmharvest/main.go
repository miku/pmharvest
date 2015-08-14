package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"launchpad.net/xmlpath"
)

var path = xmlpath.MustCompile("//resumptionToken")

type options struct {
	from   string
	prefix string
	set    string
	until  string
}

type request struct {
	link  string
	opts  options
	token string
	verb  string
}

type response struct {
	Error struct {
		Code    string `xml:"code,attr"`
		Message string `xml:",chardata"`
	} `xml:"error"`
	Payload string `xml:",innerxml"`
}

func (r request) Link() string {
	vals := url.Values{}

	MayAdd := func(k, v string) {
		if v != "" {
			vals.Add(k, v)
		}
	}

	MayAdd("from", r.opts.from)
	MayAdd("set", r.opts.set)
	MayAdd("until", r.opts.until)
	MayAdd("metadataPrefix", r.opts.prefix)
	MayAdd("verb", r.verb)
	MayAdd("resumptionToken", r.token)

	encoded := vals.Encode()
	if len(encoded) == 0 {
		return r.link
	}
	return fmt.Sprintf("%s?%s", r.link, encoded)
}

func ExtractToken(s string) string {
	root, err := xmlpath.Parse(strings.NewReader(s))
	if err != nil {
		log.Fatal(err)
	}
	if value, ok := path.String(root); ok {
		return value
	}
	return ""
}

func main() {

	link := flag.String("link", "", "OAI provider URL")
	from := flag.String("f", "2000-01-01", "from parameter")
	until := flag.String("u", time.Now().Format("2006-01-02"), "until parameter")
	prefix := flag.String("p", "oai_dc", "metadata prefix")
	set := flag.String("s", "", "set name")
	verb := flag.String("verb", "ListRecords", "OAI verb")
	verbose := flag.Bool("verbose", false, "be verbose")

	flag.Parse()

	opts := options{from: *from, until: *until, prefix: *prefix, set: *set}
	client := http.Client{}

	oair := request{opts: opts, verb: *verb, link: *link}

Loop:
	for {
		if *verbose {
			log.Println(oair.Link())
		}

		req, err := http.NewRequest("GET", oair.Link(), nil)
		if err != nil {
			log.Fatal(err)
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		defer resp.Body.Close()
		decoder := xml.NewDecoder(resp.Body)

		for {
			t, err := decoder.Token()
			if t == nil {
				break
			}
			if err != nil {
				log.Fatal(err)
			}
			switch se := t.(type) {
			case xml.StartElement:
				if se.Name.Local == "OAI-PMH" {
					var resp response
					err := decoder.DecodeElement(&resp, &se)
					if err != nil {
						log.Fatal(err)
					}
					if resp.Error.Code != "" {
						log.Fatal(resp.Error.Message)
					}
					fmt.Println(resp.Payload)
					token := ExtractToken(resp.Payload)
					if token == "" {
						break Loop
					}
					oair = request{token: token, verb: *verb, link: *link}
				}
			}
		}
	}
}
