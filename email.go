package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

type IMAP struct {
	ccc    *client.Client
	cursor uint32

	server   string
	username string
	password string
	from     string
	folder   string
	gmc      string
}

func New(server, username, password, from, folder, gmc string) (i *IMAP, err error) {
	i = &IMAP{
		ccc:    nil,
		cursor: 1,

		server:   server,
		username: username,
		password: password,
		from:     from,
		folder:   folder,
		gmc:      gmc,
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
		err = fmt.Errorf("login failed: %v", err)
		return
	}

	log.Println("Successfully logged in...")

	mbox, err := i.ccc.Select(i.folder, false)
	if err != nil {
		log.Fatal(err)
	}

	if mbox.Messages == 0 || mbox.Messages < i.cursor {
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

		if elem == nil {
			log.Fatal("Server didn't return message")
		}

		c = elem.SeqNum

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

		err = handleMessage(mr, tag, i.from, i.gmc)
		if err != nil {
			log.Fatal("Error handling message:", err)
		}

		if elem.Flags != nil && !contains(elem.Flags, imap.SeenFlag) {
			seq := new(imap.SeqSet)
			seq.AddNum(elem.SeqNum)
			item := imap.FormatFlagsOp(imap.AddFlags, true)
			flags := []interface{}{imap.SeenFlag}
			err = i.ccc.Store(seq, item, flags, nil)
			if err != nil {
				log.Println("Error marking message as read:", err)
			}
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

func contains[T comparable](strSlice []T, str T) bool {
	for _, s := range strSlice {
		if s == str {
			return true
		}
	}
	return false
}

func checkFrom(froms []*mail.Address, from string) bool {
	for _, f := range froms {
		if f.Address == from {
			return true
		}
	}
	return false
}

func handleMessage(mr *mail.Reader, tag, from, gmc string) (err error) {
	tagDir := filepath.Join(dataDir, tag)

	al, err := mr.Header.AddressList("From")
	if err != nil {
		log.Println("Error getting from:", err)
		return
	}

	if !checkFrom(al, from) {
		return
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

			codes, _ := parseCodes(filePath, gmc)
			if err != nil {
				continue
			}

			attachment := Attachment{
				Tag:      tag,
				Filename: filename,
				Codes:    codes,
			}

			err = db.Create(&attachment).Error
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	return
}

func parseCodes(filePath, gmc string) (codes []Code, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return
	}
	writer.Close()

	req, err := http.NewRequest("POST", gmc, body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to send request: %s", resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(&codes)
	if err != nil {
		return
	}

	return codes, nil
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

	fmt.Fprintf(f, "%d\n", cursor)
	i.cursor = cursor
	return nil
}

func checkMail(i *IMAP) {
	for {
		err := i.GetMessages()
		if err != nil {
			log.Println(err)
		}
		time.Sleep(time.Hour)
	}
}
