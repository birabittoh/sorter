package main

import (
	"encoding/json"
	"net/http"
)

func checkToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != token {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func getCodes(w http.ResponseWriter, r *http.Request) {
	var codes []Code
	db.Preload("Attachment").Find(&codes)
	json.NewEncoder(w).Encode(codes)
}

func getAttachments(w http.ResponseWriter, r *http.Request) {
	var attachments []Attachment
	db.Preload("Codes").Find(&attachments)
	json.NewEncoder(w).Encode(attachments)
}
