package main

import (
	"fmt"
	"os"
	"path"

	"github.com/glebarez/sqlite"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

type Task struct {
	gorm.Model
	Card   string `json:"card"`
	Amount uint   `json:"amount"`
}

type Code struct {
	gorm.Model
	AttachmentID uint   `json:"attachment_id"`
	Code         string `json:"code"`
	Value        uint   `json:"value"`
	Website      string `json:"website"`
	Done         bool   `json:"done"`

	Attachment Attachment `json:"attachment"`
}

type Attachment struct {
	gorm.Model
	Tag      string `json:"tag"`
	Filename string `json:"filename"`

	Codes []Code `json:"codes"`
}

var db *gorm.DB

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Warning: No .env file found")
	}

	if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
		panic(err)
	}

	db, err = gorm.Open(sqlite.Open(path.Join(dataDir, "data.sqlite?_pragma=foreign_keys(1)")), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	err = db.AutoMigrate(&Task{}, &Attachment{}, &Code{})
	if err != nil {
		panic(err)
	}
}

func filterAttachmentsByDone(attachments []Attachment, isDone bool) (filtered []Attachment) {
	for _, attachment := range attachments {
		if attachment.IsDone() == isDone {
			filtered = append(filtered, attachment)
		}
	}
	return filtered
}

func (a *Attachment) IsDone() bool {
	for _, code := range a.Codes {
		if !code.Done {
			return false
		}
	}
	return true
}
