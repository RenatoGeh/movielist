package main

import (
	"encoding/json"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func RegisterUser(u *tgbotapi.Update) {
	C := chat(u)
	usr := strings.ToLower(u.Message.From.UserName)
	_, e := C.allUsers[usr]
	if !e {
		C.allUsers[usr] = u.Message.From
		saveUsers(C)
	}
}

func (C *Chat) User(username string) (*tgbotapi.User, bool) {
	u, e := C.allUsers[username]
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

func saveUsers(C *Chat) {
	f, err := os.Create(C.prefix + "users.json")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer f.Close()
	b, err := json.Marshal(C.allUsers)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	_, err = f.Write(b)
	if err != nil {
		log.Printf("Error: %v", err)
	}
}

func loadUsers(C *Chat) {
	f, err := os.Open(C.prefix + "users.json")
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
	err = json.Unmarshal(b, &C.allUsers)
	if err != nil {
		log.Printf("Error: %v", err)
	}
}

func RemoveLeavers(u *tgbotapi.Update) {
	user := u.Message.LeftChatMember
	if user != nil {
		C := chat(u)
		uname := strings.ToLower(user.UserName)
		_, e := C.allUsers[uname]
		if e {
			delete(C.allUsers, uname)
			saveUsers(C)
		}
	}
}
