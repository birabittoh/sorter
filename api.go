package main

import (
	"encoding/json"
	"net/http"
)

// curl http://localhost:3000/api/codes -H "Authorization: TOKEN"
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
	err := db.Preload("Attachment").Find(&codes).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(codes)
}

func getAttachments(w http.ResponseWriter, r *http.Request) {
	var attachments []Attachment
	err := db.Preload("Codes").Find(&attachments).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(attachments)
}

func setDone(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var code Code
	err := db.First(&code, id).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	code.Done = true
	db.Save(&code)
}
