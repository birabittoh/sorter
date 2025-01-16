package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

const (
	dataDir = "data"
)

type IMAP struct {
	ccc    *client.Client
	cursor uint32

	server   string
	username string
	password string
	folder   string
}

func New(server, username, password, folder string) (i *IMAP, err error) {
	i = &IMAP{
		ccc:    nil,
		cursor: 1,

		server:   server,
		username: username,
		password: password,
		folder:   folder,
	}

	i.readCursor()
	return
}

func (i *IMAP) GetMessages() (err error) {
	i.ccc, err = client.DialTLS(i.server, nil)
	if err != nil {
		err = fmt.Errorf("TLS connection failed: %v", err)
		return
	}

	err = i.ccc.Login(i.username, i.password)
	if err != nil {
		err = fmt.Errorf("Login failed: %v", err)
		return
	}

	log.Println("Successfully logged in...")

	mbox, err := i.ccc.Select(i.folder, true)
	if err != nil {
		log.Fatal(err)
	}

	if mbox.Messages == 0 || mbox.Messages <= i.cursor {
		log.Println("There are no new messages.")
		return
	}

	log.Printf("Total message(s) in inbox: %d\n", mbox.Messages)
	seqSet := new(imap.SeqSet)
	seqSet.AddRange(i.cursor, mbox.Messages)

	var section imap.BodySectionName
	items := []imap.FetchItem{section.FetchItem()}

	messages := make(chan *imap.Message, mbox.Messages)
	go func() {
		if err := i.ccc.Fetch(seqSet, items, messages); err != nil {
			log.Fatal(err)
		}
	}()

	var c uint32
	for elem := range messages {
		// log.Println("Message no: ", elem.SeqNum)
		c = elem.SeqNum

		if elem == nil {
			log.Fatal("Server didn't return message")
		}

		r := elem.GetBody(&section)
		if r == nil {
			log.Fatal("Server didn't return message body")
		}

		mr, err := mail.CreateReader(r)
		if err != nil {
			log.Fatal(err)
		}

		var tag string
		header := mr.Header
		to, err := header.AddressList("To")
		if err != nil {
			continue // ignore emails without To
		}
		sr := strings.Split(strings.Split(to[0].Address, "@")[0], "+")
		if len(sr) < 2 {
			continue // ignore emails without tag
		}
		tag = sr[1]

		err = handleMessage(mr, tag)
		if err != nil {
			log.Fatal("Error handling message:", err)
		}
	}

	err = i.writeCursor(c)
	if err != nil {
		log.Println("Error writing cursor:", err)
	}

	err = i.ccc.Logout()
	if err != nil {
		log.Println("Failed to logout:", err)
	}

	return
}

func handleMessage(mr *mail.Reader, tag string) (err error) {
	tagDir := filepath.Join(dataDir, tag)

	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		switch h := p.Header.(type) {
		case *mail.AttachmentHeader:
			err := os.MkdirAll(tagDir, os.ModePerm)
			if err != nil {
				log.Fatal(err)
			}

			filename, err := h.Filename()
			if err != nil {
				filename = time.Now().Format(time.DateTime)
			}

			filePath := filepath.Join(tagDir, filename)
			if _, err := os.Stat(filePath); err == nil {
				log.Println("File already exists, skipping:", filename)
				continue
			}

			file, err := os.Create(filePath)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()
			io.Copy(file, p.Body)
			log.Println("Attachment:", filename)

			attachment := Attachment{
				Tag:     tag,
				Name:    filename,
				Website: "",
				Amount:  0,
				Codes:   []Code{{Code: "test"}},
			}

			err = db.Create(&attachment).Error
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	return
}

func (i *IMAP) readCursor() uint32 {
	f, err := os.Open(filepath.Join(dataDir, "cursor.txt"))
	if err != nil {
		i.cursor = 1
		return 1
	}
	defer f.Close()

	_, err = fmt.Fscanf(f, "%d", &i.cursor)
	if err != nil {
		i.cursor = 1
	}

	return i.cursor
}

func (i *IMAP) writeCursor(cursor uint32) error {
	if i.cursor == cursor {
		return nil
	}

	f, err := os.Create(filepath.Join(dataDir, "cursor.txt"))
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%d\n", cursor)
	i.cursor = cursor
	return nil
}

func getEnvDefault(key, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	return value
}

func main() {
	username, ok := os.LookupEnv("EMAIL")
	if !ok {
		log.Fatal("EMAIL var is not set")
	}
	password, ok := os.LookupEnv("PASSWORD")
	if !ok {
		log.Println("PASSWORD var is not set")
	}

	server := getEnvDefault("SERVER", "imap.gmail.com:993")
	folder := getEnvDefault("FOLDER", "INBOX")

	i, err := New(server, username, password, folder)
	if err != nil {
		log.Fatal(err)
	}

	for {
		err = i.GetMessages()
		if err != nil {
			log.Println(err)
		}
		time.Sleep(time.Hour)
	}
}
