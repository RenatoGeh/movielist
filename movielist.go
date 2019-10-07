package main

import (
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"io/ioutil"
	"log"
	"os"
	"runtime/debug"
)

const CmdHelp = "help"
const MaxGCIterations = 10

var gcIterations = 0

func Help(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	const s = "List of commands:\n" +
		"  `/all`: prints current movie list\n" +
		"  `/show i`: prints more info on the `i`-th item of list\n" +
		"  `/remove i`: removes `i`-th item from list\n" +
		"  `/add title`: adds top search result of `title` to list\n" +
		"  `/query title`: queries IMDb for `title`\n" +
		"  `/watch i1 i2 ...`: mark all `ij` instances as `watched` by you\n" +
		"  `/unwatch i1 i2 ...`: mark all `ij` instances as `unwatched` by you\n" +
		"  `/restore`: restore last automatically removed items of movie list\n" +
		"  `/watched`: prints list of watched movies\n" +
		"  `/watched username`: prints list of movies watched by username\n" +
		"  `/draw n=1`: draws n movies at random (default n=1)\n" +
		"  `/save`: force save everything\n" +
		"**Important:** before `/add`-ing, `/query` first to make sure it's the right movie!"
	msg := tgbotapi.NewMessage(u.Message.Chat.ID, s)
	msg.ReplyToMessageID = u.Message.MessageID
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
}

func loop(bot *tgbotapi.BotAPI, u *tgbotapi.Update) {
	fmt.Printf("[%s|%s] %s\n", u.Message.Chat.Title, u.Message.From.UserName, u.Message.Text)
	RegisterUser(u)
	if u.Message.IsCommand() {
		cmd := u.Message.Command()
		switch cmd {
		case CmdAll:
			log.Printf("Command /all activated")
			All(bot, u)
		case CmdShow:
			log.Printf("Command /show activated")
			Show(bot, u)
		case CmdRemove:
			log.Printf("Command /remove activated")
			Remove(bot, u)
		case CmdAdd:
			log.Printf("Command /add activated")
			Add(bot, u)
		case CmdQuery:
			log.Printf("Command /query activated")
			Query(bot, u)
		case CmdHelp:
			log.Printf("Command /help activated")
			Help(bot, u)
		case CmdWatch:
			log.Printf("Command /watch activated")
			Watch(bot, u)
		case CmdUnwatch:
			log.Printf("Command /unwatch activated")
			Unwatch(bot, u)
		case CmdRestore:
			log.Printf("Command /restore activated")
			Restore(bot, u)
		case CmdWatched:
			log.Printf("Command /watched activated")
			Watched(bot, u)
		case CmdDraw:
			log.Printf("Command /draw activated")
			Draw(bot, u)
		case CmdSave:
			log.Printf("Command /save activated")
			Save(bot, u)
		}
	}
	gcIterations++
	if gcIterations%MaxGCIterations == 0 {
		debug.FreeOSMemory()
	}
}

func getToken() string {
	f, err := os.Open("token.tk")
	if err != nil {
		log.Panic(err)
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Panic(err)
	}
	return string(b[:len(b)-1])
}

func main() {
	bot, err := tgbotapi.NewBotAPI(getToken())
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	loadMovies()
	loadUsers()
	for update := range updates {
		if update.Message == nil {
			continue
		}
		loop(bot, &update)
	}
}
