package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

const dataDir = "data"

var token string

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
	token, ok = os.LookupEnv("TOKEN")
	if !ok {
		log.Fatal("TOKEN var is not set")
	}
	from, ok := os.LookupEnv("FROM")
	if !ok {
		log.Fatal("FROM var is not set")
	}

	server := getEnvDefault("SERVER", "imap.gmail.com:993")
	folder := getEnvDefault("FOLDER", "INBOX")
	gmc := getEnvDefault("GMC_INSTANCE", "http://gmc:8080/")

	i, err := New(server, username, password, from, folder, gmc)
	if err != nil {
		log.Fatal(err)
	}

	go checkMail(i)

	// API
	r := http.NewServeMux()
	r.HandleFunc("GET /api/codes", checkToken(getCodes))
	r.HandleFunc("GET /api/attachments", checkToken(getAttachments))

	s := &http.Server{
		Addr:         ":3000",
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,

		// For debugging
		ErrorLog: log.New(os.Stderr, "", 0),
	}

	log.Println("Starting server on", s.Addr)
	log.Fatal(s.ListenAndServe())
}
