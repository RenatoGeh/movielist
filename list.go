package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"sort"
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
	last_query	   string
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
	CmdDraw    = "draw"
	CmdSave    = "save"
	CmdRanking = "ranking"
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
	if query == "" {
		query = last_query
	}
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
	if err != nil || (n < maxImageSize && img.Bounds().Max.X <= 1920) {
		byFile = false
		goto send
	}
	log.Printf("Compressing cover...")
	for n > maxImageSize || img.Bounds().Max.X > 1920 {
		log.Printf("  %d/%d", n, maxImageSize)
		img, n, cbytes, err = Resize(img)
		log.Printf("  -- %d/%d", n, maxImageSize)
	}
	if err != nil {
		log.Printf("Error: %v", err)
		byFile = false
		goto send
	}
	icover = tgbotapi.FileBytes{"cover", cbytes}
send:
	log.Printf("Image has bounds: %v", img.Bounds())
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
	last_query = q
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
	r := movies[i]
	movies = append(movies[:i], movies[i+1:]...)
	s := fmt.Sprintf("Removing %s (%d) from movie list...", r.Title, r.Year)
	msg := tgbotapi.NewMessage(u.Message.Chat.ID, s)
	msg.ReplyToMessageID = u.Message.MessageID
	bot.Send(msg)
	saveMovies()
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
	saveMovies()
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
		saveMovies()
	}
}

func Watched(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	var s string
	if u.Message.CommandArguments() != "" {
		uname := ToUsername(u)
		_, e := User(uname)
		if !e {
			s = fmt.Sprintf("I don't know who %s is!", uname)
			goto send
		}
		s = fmt.Sprintf("Movies watched by %s still in the to-watch list:\n", uname)
		var c int
		for i, m := range movies {
			for _, w := range m.WatchedBy {
				if strings.ToLower(w) == uname {
					s += fmt.Sprintf("  %d. %s (%d) {%d}\n", c, m.Title, m.Year, i)
					c++
					break
				}
			}
		}
		s += fmt.Sprintf("Movies watched by %s in the watched list:\n", uname)
		var d int
		for _, m := range watched_movies {
			for _, w := range m.WatchedBy {
				if strings.ToLower(w) == uname {
					s += fmt.Sprintf("  %d. %s (%d)\n", d, m.Title, m.Year)
					d++
					break
				}
			}
		}
		s += fmt.Sprintf("Total movies watched: %d", c+d)
		goto send
	}
	if len(watched_movies) == 0 {
		s = "You have not watched any movies yet! :("
	} else {
		s = "Watched movie list:\n"
		for i, m := range watched_movies {
			s += fmt.Sprintf("  %d. %s (%d)\n", i, m.Title, m.Year)
		}
	}
send:
	msg := tgbotapi.NewMessage(u.Message.Chat.ID, s)
	msg.ReplyToMessageID = u.Message.MessageID
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
}

func Draw(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	args := u.Message.CommandArguments()
	var n int
	if args != "" {
		L, err := extractIndices(args)
		if err != nil {
			log.Printf("Error: %v", err)
			return
		}
		n = L[0]
	} else {
		n = 1
	}
	if n > len(movies) {
		n = len(movies)
	}
	type entryIndex struct {
		e Entry
		i int
	}
	var M []entryIndex
	G := make(map[int]bool)
	for i := 0; i < n; i++ {
		for {
			k := rand.Intn(len(movies))
			if _, e := G[k]; !e {
				G[k] = true
				M = append(M, entryIndex{movies[k], k})
				break
			}
		}
	}
	log.Println(M)
	if M != nil {
		s := "I've chosen these movies for you to watch. Have fun! :)\n"
		for i, m := range M {
			s += fmt.Sprintf("  %d. %s (%d) {%d}\n", i, m.e.Title, m.e.Year, m.i)
		}
		s += "You can find out more about each movie with `/show i` where `i` is the number in " +
			"{curly braces}. Don't forget to `/watch i` when you're finished watching movie `i`!"
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, s)
		msg.ReplyToMessageID = u.Message.MessageID
		msg.ParseMode = tgbotapi.ModeMarkdown
		bot.Send(msg)
	}
}

func Save(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	saveMovies()
	saveUsers()
}

func saveList(filename string, list []Entry) {
	f, err := os.Create(filename)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer f.Close()
	b, err := json.Marshal(list)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	_, err = f.Write(b)
	if err != nil {
		log.Printf("Error: %v", err)
	}
}

func saveMovies() {
	saveList("movies.json", movies)
	saveList("watched.json", watched_movies)
	saveList("undo.json", undo_movies)
}

func loadList(filename string, list *[]Entry) {
	f, err := os.Open(filename)
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
	err = json.Unmarshal(b, list)
	if err != nil {
		log.Printf("Error: %v", err)
	}
}

func loadMovies() {
	loadList("movies.json", &movies)
	loadList("watched.json", &watched_movies)
	loadList("undo.json", &undo_movies)
}

func Ranking(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	type stats struct {
		u *tgbotapi.User
		w int
	}
	M := make(map[string]*stats)
	for s, u := range allUsers {
		M[strings.ToLower(s)] = &stats{u, 0}
	}
	for _, m := range movies {
		for _, w := range m.WatchedBy {
			if s, e := M[strings.ToLower(w)]; e {
				s.w++
			}
		}
	}
	for _, m := range watched_movies {
		for _, w := range m.WatchedBy {
			if s, e := M[strings.ToLower(w)]; e {
				s.w++
			}
		}
	}
	n := len(M)
	S := make([]*stats, n)
	var i int
	for _, s := range M {
		S[i] = s
		i++
	}
	sort.Slice(S, func(i, j int) bool {
		return S[i].w > S[j].w
	})
	t := "Ranking of number of watched movies:\n"
	for i, s := range S {
		t += fmt.Sprintf("  %d. %s (%d)\n", i+1, s.u.UserName, s.w)
	}
	msg := tgbotapi.NewMessage(u.Message.Chat.ID, t)
	msg.ReplyToMessageID = u.Message.MessageID
	bot.Send(msg)
}
