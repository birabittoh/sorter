package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const dataDir = "data"

var token string

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

	minutesStr := getEnvDefault("CHECK_INTERVAL", "15")
	minutes, err := strconv.ParseUint(minutesStr, 10, 64)
	if err != nil {
		minutes = 15
	}

	server := getEnvDefault("SERVER", "imap.gmail.com:993")
	folder := getEnvDefault("FOLDER", "INBOX")
	gmc := getEnvDefault("GMC_INSTANCE", "http://localhost:5000/")
	addr := getEnvDefault("ADDR", ":3000")

	i, err := New(server, username, password, from, folder, gmc)
	if err != nil {
		log.Fatal(err)
	}

	go checkMail(i, time.Duration(minutes)*time.Minute)

	// API
	r := http.NewServeMux()
	r.HandleFunc("GET /api/codes", checkToken(getCodes))
	r.HandleFunc("GET /api/attachments", checkToken(getAttachments))
	r.HandleFunc("GET /api/tags", checkToken(getTags))
	r.HandleFunc("GET /api/tasks", checkToken(getTasks))
	r.HandleFunc("POST /api/attachments/{id}", checkToken(setAttachment))
	r.HandleFunc("POST /api/codes/{id}", checkToken(setCode))
	r.HandleFunc("POST /api/tasks", checkToken(newTask))

	s := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,

		// For debugging
		ErrorLog: log.New(os.Stderr, "", 0),
	}

	log.Println("Starting server on", s.Addr)
	log.Fatal(s.ListenAndServe())
}
