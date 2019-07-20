package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

type Entry struct {
	Title string
	Year  int
	Cover string
	ID    string
}

var movies []Entry

const CmdAll = "all"
const CmdShow = "show"
const CmdRemove = "remove"
const CmdAdd = "add"
const CmdQuery = "query"

const maxImageSize = 5000000

const imdbPreamble = "https://www.imdb.com/title/"

func containsMovie(e *Entry) bool {
	for _, m := range movies {
		if e.Title == m.Title && e.Year == m.Year {
			return true
		}
	}
	return false
}

func AddEntry(e *Entry) int {
	if !containsMovie(e) {
		movies = append(movies, *e)
		saveMovies()
		return len(movies) - 1
	} else {
		return -1
	}
}

func Add(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	query := u.Message.CommandArguments()
	e := Retrieve(query)
	if e == nil {
		return
	}
	if AddEntry(e) < 0 {
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, "Movie is already in our to-watch list!")
		msg.ReplyToMessageID = u.Message.MessageID
		bot.Send(msg)
		return
	}
	preview(bot, u, e)
}

func All(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	var s string
	if len(movies) == 0 {
		s = "Movie list is empty! Start adding movies with /add!"
	} else {
		s = "To-watch movie list:\n"
		for i, m := range movies {
			s += fmt.Sprintf("  %d. %s (%d)\n", i, m.Title, m.Year)
		}
		s += "`/show i` - shows more information on the `i`-th movie."
	}
	msg := tgbotapi.NewMessage(u.Message.Chat.ID, s)
	msg.ReplyToMessageID = u.Message.MessageID
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
}

func getMovie(s string) (int, *Entry) {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("Error: %v", err)
		return 0, nil
	}
	if i < 0 || i >= len(movies) {
		return 0, nil
	}
	return i, &movies[i]
}

func preview(bot *tgbotapi.BotAPI, u *tgbotapi.Update, m *Entry) {
	if m == nil {
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, "Could not find requested query!")
		msg.ReplyToMessageID = u.Message.MessageID
		bot.Send(msg)
		return
	}
	var cbytes []byte
	var icover tgbotapi.FileBytes
	scover := m.Cover
	byFile := true
	img, n, err := GetImage(m.Cover)
	if err != nil || n < maxImageSize {
		byFile = false
		goto send
	}
	log.Printf("Compressing cover...")
	for n > maxImageSize {
		log.Printf("  %d/%d", n, maxImageSize)
		img, n = Resize(img)
	}
	cbytes, err = Encode(img)
	if err != nil {
		log.Printf("Error: %v", err)
		byFile = false
		goto send
	}
	icover = tgbotapi.FileBytes{"cover", cbytes}
send:
	var msg tgbotapi.PhotoConfig
	if byFile {
		log.Printf("Sending cover by file.")
		msg = tgbotapi.NewPhotoUpload(u.Message.Chat.ID, icover)
	} else {
		log.Printf("Sending cover by URL.")
		msg = tgbotapi.NewPhotoShare(u.Message.Chat.ID, scover)
	}
	msg.Caption = fmt.Sprintf("%s (%d)\nIMDb: %s%s", m.Title, m.Year, imdbPreamble, m.ID)
	msg.ReplyToMessageID = u.Message.MessageID
	bot.Send(msg)
}

func Query(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	q := u.Message.CommandArguments()
	preview(bot, u, Retrieve(q))
}

func Show(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	if len(movies) == 0 {
		return
	}
	_, m := getMovie(u.Message.CommandArguments())
	if m == nil {
		return
	}
	preview(bot, u, m)
}

func Remove(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	i, m := getMovie(u.Message.CommandArguments())
	if m == nil {
		return
	}
	movies = append(movies[0:i], movies[i+1:len(movies)]...)
	s := fmt.Sprintf("Removing %s (%d) from movie list...", m.Title, m.Year)
	msg := tgbotapi.NewMessage(u.Message.Chat.ID, s)
	msg.ReplyToMessageID = u.Message.MessageID
	bot.Send(msg)
}

func saveMovies() {
	f, err := os.Create("movies.json")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer f.Close()
	b, err := json.Marshal(movies)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	_, err = f.Write(b)
	if err != nil {
		log.Printf("Error: %v", err)
	}
}

func loadMovies() {
	f, err := os.Open("movies.json")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer f.Close()
	s, err := f.Stat()
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	if s.Size() < 5 {
		return
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	err = json.Unmarshal(b, &movies)
	if err != nil {
		log.Printf("Error: %v", err)
	}
}
