package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

type Entry struct {
	Title     string
	Year      int
	Cover     string
	ID        string
	WatchedBy []string
}

var (
	movies         []Entry
	undo_movies    []Entry
	watched_movies []Entry
)

const (
	CmdAll     = "all"
	CmdShow    = "show"
	CmdRemove  = "remove"
	CmdAdd     = "add"
	CmdQuery   = "query"
	CmdWatch   = "watch"
	CmdUnwatch = "unwatch"
	CmdRestore = "restore"
	CmdWatched = "watched"
)

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
	if len(m.WatchedBy) != 0 {
		msg.Caption += fmt.Sprintf("\nWatched by (%d):", len(m.WatchedBy))
		for _, usr := range m.WatchedBy {
			msg.Caption += fmt.Sprintf(" @%s", usr)
		}
	}
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
	movies = append(movies[:i], movies[i+1:]...)
	s := fmt.Sprintf("Removing %s (%d) from movie list...", m.Title, m.Year)
	msg := tgbotapi.NewMessage(u.Message.Chat.ID, s)
	msg.ReplyToMessageID = u.Message.MessageID
	bot.Send(msg)
}

func extractIndices(whole string) ([]int, error) {
	args := strings.Fields(whole)
	L := make([]int, len(args))
	for i, s := range args {
		var err error
		L[i], err = strconv.Atoi(s)
		if err != nil {
			log.Printf("Error: %v", err)
			return nil, err
		}
	}
	return L, nil
}

func checkWatched() string {
	var msg string
	var nlist []Entry
	undo_movies = []Entry{}
	for _, m := range movies {
		if len(m.WatchedBy) >= len(allUsers) {
			msg += fmt.Sprintf("  %s (%d)\n", m.Title, m.Year)
			undo_movies = append(undo_movies, m)
			watched_movies = append(watched_movies, m)
		} else {
			nlist = append(nlist, m)
		}
	}
	movies = nlist
	if msg != "" {
		msg = fmt.Sprintf("I've removed the following movies because everyone has watched them!\n%s"+
			"To undo these changes, tell me to `/restore`.", msg)
	}
	return msg
}

func Watch(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	usr := u.Message.From.UserName
	W, err := extractIndices(u.Message.CommandArguments())
	if err != nil {
		return
	}
	var change bool
	for _, w := range W {
		if w >= 0 && w < len(movies) {
			var in bool
			L := movies[w].WatchedBy
			for _, name := range L {
				if name == usr {
					in = true
					break
				}
			}
			if !in {
				movies[w].WatchedBy = append(movies[w].WatchedBy, usr)
				change = true
			}
		}
	}
	if change {
		if c := checkWatched(); c != "" {
			msg := tgbotapi.NewMessage(u.Message.Chat.ID, c)
			msg.ReplyToMessageID = u.Message.MessageID
			msg.ParseMode = tgbotapi.ModeMarkdown
			bot.Send(msg)
		}
	}
}

func Unwatch(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	usr := u.Message.From.UserName
	W, err := extractIndices(u.Message.CommandArguments())
	if err != nil {
		return
	}
	for _, w := range W {
		if w >= 0 && w < len(movies) {
			m := &movies[w]
			i := -1
			for j, watcher := range m.WatchedBy {
				if watcher == usr {
					i = j
					break
				}
			}
			if i >= 0 {
				m.WatchedBy = append(m.WatchedBy[:i], m.WatchedBy[i+1:]...)
			}
		}
	}
}

func Restore(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	if undo_movies != nil {
		for _, m := range undo_movies {
			m.WatchedBy = []string{}
			movies = append(movies, m)
		}
		watched_movies = append(watched_movies[:len(watched_movies)-len(undo_movies)])
		undo_movies = nil
	}
}

func Watched(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	var s string
	if len(watched_movies) == 0 {
		s = "You have not watched any movies yet! :("
	} else {
		s = "Watched movie list:\n"
		for i, m := range watched_movies {
			s += fmt.Sprintf("  %d. %s (%d)\n", i, m.Title, m.Year)
		}
	}
	msg := tgbotapi.NewMessage(u.Message.Chat.ID, s)
	msg.ReplyToMessageID = u.Message.MessageID
	msg.ParseMode = tgbotapi.ModeMarkdown
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
