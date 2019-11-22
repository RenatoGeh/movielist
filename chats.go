package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"io/ioutil"
	"log"
	"os"
)

type Chat struct {
	prefix        string
	movies        []Entry
	undoMovies    []Entry
	watchedMovies []Entry
	lastQuery     string
	allUsers      map[string]*tgbotapi.User
}

var chatMap map[int64]*Chat = make(map[int64]*Chat)

func chat(u *tgbotapi.Update) *Chat {
	id := u.Message.Chat.ID
	var C *Chat
	var e bool
	if C, e = chatMap[id]; !e {
		C = &Chat{fmt.Sprintf("chat%d/", id), nil, nil, nil, "", make(map[string]*tgbotapi.User)}
		var newChat bool
		if _, err := os.Stat(C.prefix); os.IsNotExist(err) {
			err = os.Mkdir(C.prefix, os.ModePerm)
			if err != nil {
				log.Printf("Error: %v", err)
			}
			newChat = true
		} else {
			loadList(C.prefix+"movies.json", &C.movies)
			loadList(C.prefix+"watched.json", &C.watchedMovies)
			loadList(C.prefix+"undo.json", &C.undoMovies)
			loadUsers(C)
		}
		chatMap[id] = C
		if newChat {
			saveChats()
		}
	}
	return C
}

func saveChats() {
	f, err := os.Create("chats.json")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer f.Close()
	b, err := json.Marshal(chatMap)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	_, err = f.Write(b)
	if err != nil {
		log.Printf("Error: %v", err)
	}
}

func loadChats() {
	f, err := os.Open("chats.json")
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
	err = json.Unmarshal(b, &chatMap)
	if err != nil {
		log.Printf("Error: %v", err)
	}
}
