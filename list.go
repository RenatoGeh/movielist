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

const imdbPreamble = "https://www.imdb.com/title/"

func AddEntry(e *Entry) int {
	movies = append(movies, *e)
	saveMovies()
	return len(movies) - 1
}

func Add(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	query := u.Message.CommandArguments()
	e := Retrieve(query)
	if e == nil {
		return
	}
	AddEntry(e)
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
	return i, &movies[i]
}

func preview(bot *tgbotapi.BotAPI, u *tgbotapi.Update, m *Entry) {
	c := fmt.Sprintf("%s (%d)\nIMDb: %s%s", m.Title, m.Year, imdbPreamble, m.ID)
	msg := tgbotapi.NewPhotoShare(u.Message.Chat.ID, m.Cover)
	msg.Caption = c
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
