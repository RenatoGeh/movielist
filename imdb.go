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
	if query == nil || query["d"] == nil {
		return nil
	}
	log.Printf("Fetching JSON...\n%v", query)
	entries := query["d"].([]interface{})
	for i := range entries {
		e := entries[i].(map[string]interface{})
		title := e[titleKey].(string)
		log.Printf("Title: %s", title)
		if _, exists := e[yearKey]; !exists {
			continue
		}
		year := int(e[yearKey].(float64))
		log.Printf("Year: %d", year)
		if e[coverKey] == nil {
			continue
		}
		cover := e[coverKey].([]interface{})[0].(string)
		log.Printf("Cover URL: %s", cover)
		id := e[idKey].(string)
		log.Printf("IMDb ID: %s", id)
		return &Entry{title, year, cover, id, []string{}}
	}
	return nil
}

// Retrieve returns an Entry from IMDb's Search Suggestions API.
func Retrieve(query string) *Entry {
	q := ascii(query)
	if q == "" {
		return nil
	}
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
