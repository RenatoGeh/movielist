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

func containsMovie(e *Entry, c []Entry) bool {
	for _, m := range c {
		if e.Title == m.Title && e.Year == m.Year {
			return true
		}
	}
	return false
}

func AddEntry(e *Entry, u *tgbotapi.Update) int {
	if C := chat(u); !containsMovie(e, C.movies) {
		C.movies = append(C.movies, *e)
		saveMovies(C)
		return len(C.movies) - 1
	} else {
		return -1
	}
}

func Add(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	C := chat(u)
	query := u.Message.CommandArguments()
	if query == "" {
		query = C.lastQuery
	}
	e := Retrieve(query)
	if e == nil {
		return
	}
	if AddEntry(e, u) < 0 {
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, "Movie is already in our to-watch list!")
		msg.ReplyToMessageID = u.Message.MessageID
		bot.Send(msg)
		return
	}
	preview(bot, u, e)
}

func All(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	var s string
	C := chat(u)
	if len(C.movies) == 0 {
		s = "Movie list is empty! Start adding movies with /add!"
	} else {
		s = "To-watch movie list:\n"
		for i, m := range C.movies {
			s += fmt.Sprintf("  %d. %s (%d)\n", i, m.Title, m.Year)
		}
		s += "`/show i` - shows more information on the `i`-th movie."
	}
	msg := tgbotapi.NewMessage(u.Message.Chat.ID, s)
	msg.ReplyToMessageID = u.Message.MessageID
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
}

func getMovie(s string, C *Chat) (int, *Entry) {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("Error: %v", err)
		return 0, nil
	}
	if i < 0 || i >= len(C.movies) {
		return 0, nil
	}
	return i, &C.movies[i]
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
	turl := imdbPreamble + m.ID
	msg.Caption = fmt.Sprintf("%s (%d)\nRating: %.1f/10.0\nIMDb: %s", m.Title, m.Year, Rating(turl), turl)
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
	C := chat(u)
	C.lastQuery = q
	preview(bot, u, Retrieve(q))
}

func Show(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	C := chat(u)
	if len(C.movies) == 0 {
		return
	}
	_, m := getMovie(u.Message.CommandArguments(), C)
	if m == nil {
		return
	}
	preview(bot, u, m)
}

func Remove(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	C := chat(u)
	i, m := getMovie(u.Message.CommandArguments(), C)
	if m == nil {
		return
	}
	r := C.movies[i]
	C.movies = append(C.movies[:i], C.movies[i+1:]...)
	s := fmt.Sprintf("Removing %s (%d) from movie list...", r.Title, r.Year)
	msg := tgbotapi.NewMessage(u.Message.Chat.ID, s)
	msg.ReplyToMessageID = u.Message.MessageID
	bot.Send(msg)
	saveMovies(C)
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

func checkWatched(u *tgbotapi.Update) string {
	var msg string
	var nlist []Entry
	C := chat(u)
	C.undoMovies = []Entry{}
	for _, m := range C.movies {
		if len(m.WatchedBy) >= len(C.allUsers) {
			msg += fmt.Sprintf("  %s (%d)\n", m.Title, m.Year)
			C.undoMovies = append(C.undoMovies, m)
			C.watchedMovies = append(C.watchedMovies, m)
		} else {
			nlist = append(nlist, m)
		}
	}
	C.movies = nlist
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
	C := chat(u)
	for _, w := range W {
		if w >= 0 && w < len(C.movies) {
			var in bool
			L := C.movies[w].WatchedBy
			for _, name := range L {
				if name == usr {
					in = true
					break
				}
			}
			if !in {
				C.movies[w].WatchedBy = append(C.movies[w].WatchedBy, usr)
				change = true
			}
		}
	}
	if change {
		if c := checkWatched(u); c != "" {
			msg := tgbotapi.NewMessage(u.Message.Chat.ID, c)
			msg.ReplyToMessageID = u.Message.MessageID
			msg.ParseMode = tgbotapi.ModeMarkdown
			bot.Send(msg)
		}
	}
	saveMovies(C)
}

func Unwatch(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	usr := u.Message.From.UserName
	W, err := extractIndices(u.Message.CommandArguments())
	if err != nil {
		return
	}
	C := chat(u)
	for _, w := range W {
		if w >= 0 && w < len(C.movies) {
			m := &C.movies[w]
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
	if C := chat(u); C.undoMovies != nil {
		for _, m := range C.undoMovies {
			m.WatchedBy = []string{}
			C.movies = append(C.movies, m)
		}
		C.watchedMovies = append(C.watchedMovies[:len(C.watchedMovies)-len(C.undoMovies)])
		C.undoMovies = nil
		saveMovies(C)
	}
}

func Watched(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	var s string
	C := chat(u)
	if u.Message.CommandArguments() != "" {
		uname := ToUsername(u)
		_, e := C.User(uname)
		if !e {
			s = fmt.Sprintf("I don't know who %s is!", uname)
			goto send
		}
		s = fmt.Sprintf("Movies watched by %s still in the to-watch list:\n", uname)
		var c int
		for i, m := range C.movies {
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
		for _, m := range C.watchedMovies {
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
	if len(C.watchedMovies) == 0 {
		s = "You have not watched any movies yet! :("
	} else {
		s = "Watched movie list:\n"
		for i, m := range C.watchedMovies {
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
	C := chat(u)
	if n > len(C.movies) {
		n = len(C.movies)
	}
	type entryIndex struct {
		e Entry
		i int
	}
	var M []entryIndex
	G := make(map[int]bool)
	for i := 0; i < n; i++ {
		for {
			k := rand.Intn(len(C.movies))
			if _, e := G[k]; !e {
				G[k] = true
				M = append(M, entryIndex{C.movies[k], k})
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
	C := chat(u)
	saveMovies(C)
	saveUsers(C)
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

func saveMovies(C *Chat) {
	saveList(C.prefix+"movies.json", C.movies)
	saveList(C.prefix+"watched.json", C.watchedMovies)
	saveList(C.prefix+"undo.json", C.undoMovies)
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

func loadMovies(u *tgbotapi.Update) {
	C := chat(u)
	loadList(C.prefix+"movies.json", &C.movies)
	loadList(C.prefix+"watched.json", &C.watchedMovies)
	loadList(C.prefix+"undo.json", &C.undoMovies)
}

func Ranking(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	type stats struct {
		u *tgbotapi.User
		w int
	}
	M := make(map[string]*stats)
	C := chat(u)
	for s, u := range C.allUsers {
		M[strings.ToLower(s)] = &stats{u, 0}
	}
	for _, m := range C.movies {
		for _, w := range m.WatchedBy {
			if s, e := M[strings.ToLower(w)]; e {
				s.w++
			}
		}
	}
	for _, m := range C.watchedMovies {
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
