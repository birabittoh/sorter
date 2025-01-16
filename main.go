package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/joho/godotenv"
)

const (
	dataDir = "data"
)

func getEnvDefault(key, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	return value
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

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

	ccc, err := client.DialTLS(server, nil)
	if err != nil {
		log.Fatalf("TLS failed: %v\n", err)
	}

	err = ccc.Login(username, password)
	if err != nil {
		log.Println("Failed to login:", err)
	} else {
		log.Println("Succesfully logged in...")
	}

	mbox, err := ccc.Select(folder, false)
	if err != nil {
		log.Fatal(err)
	}

	if mbox.Messages == 0 {
		log.Fatal("No messages in mailbox")
	} else {
		log.Printf("Total message(s) in inbox: %d\n", mbox.Messages)
	}
	seqSet := new(imap.SeqSet)
	seqSet.AddRange(1, mbox.Messages)

	var section imap.BodySectionName
	items := []imap.FetchItem{section.FetchItem()}

	messages := make(chan *imap.Message, mbox.Messages)
	go func() {
		if err := ccc.Fetch(seqSet, items, messages); err != nil {
			log.Fatal(err)
		}
	}()

	for elem := range messages {
		log.Println("Message no: ", elem.SeqNum)

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
		tagDir := filepath.Join(dataDir, tag)

		if subject, err := header.Subject(); err == nil {
			log.Println("Subject:", subject)
		}
		if date, err := header.Date(); err == nil {
			log.Println("Date:", date)
		}
		if from, err := header.AddressList("From"); err == nil {
			log.Println("From:", from)
		}

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

				file, err := os.Create(filepath.Join(tagDir, filename))
				if err != nil {
					log.Fatal(err)
				}
				defer file.Close()
				io.Copy(file, p.Body)
				log.Println("Attachment:", filename)
			}
		}

	}

	if err := ccc.Logout(); err != nil {
		log.Printf("Failed to logout: %v\n", err)
	}
}
