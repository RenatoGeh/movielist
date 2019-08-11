package main

import (
	"encoding/json"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var allUsers = make(map[string]*tgbotapi.User)

func RegisterUser(u *tgbotapi.Update) {
	usr := strings.ToLower(u.Message.From.UserName)
	_, e := allUsers[usr]
	if !e {
		allUsers[usr] = u.Message.From
		saveUsers()
	}
}

func User(username string) (*tgbotapi.User, bool) {
	u, e := allUsers[username]
	return u, e
}

func ToUsername(u *tgbotapi.Update) string {
	s := strings.ToLower(strings.TrimSpace(u.Message.CommandArguments()))
	if s != "" {
		if s[0] == '@' {
			s = s[1:]
		}
	}
	return s
}

func saveUsers() {
	f, err := os.Create("users.json")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer f.Close()
	b, err := json.Marshal(allUsers)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	_, err = f.Write(b)
	if err != nil {
		log.Printf("Error: %v", err)
	}
}

func loadUsers() {
	f, err := os.Open("users.json")
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
	err = json.Unmarshal(b, &allUsers)
	if err != nil {
		log.Printf("Error: %v", err)
	}
}
