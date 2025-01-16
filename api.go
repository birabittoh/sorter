package main

import (
	"encoding/json"
	"net/http"
)

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
