package main

import (
	"net/http"

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

func setCode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	r.ParseForm()

	var changed bool
	query := db.Model(&Code{}).Where("id = ?", id).Session(&gorm.Session{})

	done := r.Form.Get("done")
	if done != "" {
		err := query.Update("done", done).Error
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
