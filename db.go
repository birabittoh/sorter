package main

import (
	"fmt"
	"os"
	"path"

	"github.com/glebarez/sqlite"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

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

	err = db.AutoMigrate(&Attachment{}, &Code{})
	if err != nil {
		panic(err)
	}
}
