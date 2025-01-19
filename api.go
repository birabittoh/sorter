package main

import (
	"encoding/json"
	"net/http"
	"time"

	"gorm.io/gorm"
)

// curl http://localhost:3000/api/codes -H "Authorization: TOKEN"
func checkToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != token {
			jsonError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		next(w, r)
	}
}

func getCodes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	tag := q.Get("tag")
	done := q.Get("done")

	query := db.Model(&Code{}).Preload("Attachment").Joins("JOIN attachments ON codes.attachment_id = attachments.id")

	if tag != "" {
		query = query.Where("attachments.tag = ?", tag)
	}
	if done != "" {
		query = query.Where("codes.done = ?", parseBool(done))
	}

	var codes []Code
	err := query.Find(&codes).Error
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, codes)
}

func getAttachments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	tag := q.Get("tag")
	done := q.Get("done")

	var attachments []Attachment
	query := db.Preload("Codes")
	if tag != "" {
		query = query.Where(&Attachment{Tag: tag})
	}

	err := query.Find(&attachments).Error
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if done != "" {
		attachments = filterAttachmentsByDone(attachments, parseBool(done))
	}

	jsonResponse(w, http.StatusOK, attachments)
}

func getTags(w http.ResponseWriter, r *http.Request) {
	var tags []string
	err := db.Model(&Attachment{}).Distinct("tag").Pluck("tag", &tags).Error
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, tags)
}

func getTasks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	today := q.Get("today")
	card := q.Get("card")

	query := db.Order("created_at DESC")
	if parseBool(today) { // today=true if today=""
		today := time.Now().Truncate(24 * time.Hour)
		tomorrow := today.Add(24 * time.Hour)
		query = query.Where("created_at >= ? AND created_at < ?", today, tomorrow)
	}
	if card != "" {
		query = query.Where("card = ?", card)
	}

	var tasks []Task
	err := query.Find(&tasks).Error
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, tasks)
}

func setCode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := r.ParseForm()
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	var changed bool
	query := db.Model(&Code{}).Where("id = ?", id).Session(&gorm.Session{})

	done := r.Form.Get("done")
	if done != "" {
		err := query.Update("done", parseBool(done)).Error
		if err != nil {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		changed = true
	}

	if changed {
		jsonResponse(w, http.StatusAccepted, nil)
		return
	}

	jsonError(w, http.StatusNotFound, "nothing to do")
}

func newTask(w http.ResponseWriter, r *http.Request) {
	var task Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if task.Card == "" || task.Amount == 0 {
		jsonError(w, http.StatusBadRequest, "Card and amount are required")
		return
	}

	err = db.Create(&task).Error
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusCreated, task)
}
