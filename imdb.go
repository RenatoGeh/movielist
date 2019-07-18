package main

import (
	"encoding/json"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"unicode"
)

const apiPreamble = "https://sg.media-imdb.com/suggests/"

const (
	titleKey = "l"
	idKey    = "id"
	starsKey = "s"
	yearKey  = "y"
	coverKey = "i"
)

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r)
}

func ascii(s string) string {
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	r, _, _ := transform.String(t, s)
	return r
}

// convert converts a JSON-P string into an Entry.
func convert(cnt string) *Entry {
	i := strings.Index(cnt, "(")
	cnt = cnt[i+1 : len(cnt)-1]

	var query map[string]interface{}
	err := json.Unmarshal([]byte(cnt), &query)
	if err != nil {
		log.Printf("Error: %v", err)
		return nil
	}
	entries := query["d"].([]interface{})
	first := entries[0].(map[string]interface{})

	title := first[titleKey].(string)
	year := int(first[yearKey].(float64))
	cover := first[coverKey].([]interface{})[0].(string)
	id := first[idKey].(string)
	return &Entry{title, year, cover, id}
}

// Retrieve returns an Entry from IMDb's Search Suggestions API.
func Retrieve(query string) *Entry {
	q := ascii(query)
	url := apiPreamble + strings.ToLower(string(q[0])) + "/" + url.PathEscape(query+".json")
	r, err := http.Get(url)
	if err != nil {
		log.Printf("Error: %v", err)
		return nil
	}
	defer r.Body.Close()
	cnt, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error: %v", err)
		return nil
	}
	return convert(string(cnt))
}
