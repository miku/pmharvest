package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

type options struct {
	from   string
	link   string
	prefix string
	set    string
	until  string
}

type request struct {
	opts  options
	token string
	verb  string
}

type response struct {
	Date    string `xml:"responseDate"`
	Request struct {
		Verb  string `xml:"verb,attr"`
		Set   string `xml:"set,attr"`
		From  string `xml:"from,attr"`
		Until string `xml:"until,attr"`
		Link  string `xml:",chardata"`
	} `xml:"request"`
	Error struct {
		Code    string `xml:"code,attr"`
		Message string `xml:",chardata"`
	} `xml:"error"`
	ListRecords string `xml:",innerxml"`
}

func (r request) Link() string {
	v := url.Values{}
	v.Add("from", r.opts.from)
	v.Add("set", r.opts.set)
	v.Add("until", r.opts.until)
	v.Add("metadataPrefix", r.opts.prefix)
	v.Add("verb", r.verb)
	if r.token != "" {
		v.Add("resumptionToken", r.token)
	}
	return fmt.Sprintf("%s?%s", r.opts.link, v.Encode())
}

func main() {

	link := flag.String("link", "", "OAI provider URL")
	// output := flag.String("o", "", "output file")
	from := flag.String("f", "2000-01-01", "from parameter")
	until := flag.String("u", time.Now().Format("2006-01-02"), "until parameter")
	prefix := flag.String("p", "oai_dc", "metadata prefix")
	set := flag.String("s", "", "set name")
	verbose := flag.Bool("verbose", false, "be verbose")

	flag.Parse()

	opts := options{from: *from, until: *until, prefix: *prefix, set: *set, link: *link}
	oair := request{opts: opts, verb: "ListRecords"}

	client := http.Client{}

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
				fmt.Printf("%+v", resp)
			}
		}
	}
}
