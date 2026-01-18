package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

type ContactTestHelper struct {
	Router    http.Handler
	AuthToken string
}

func (h *ContactTestHelper) SubmitContact(reqBody interface{}) (*http.Response, []byte, error) {
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/contact", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Router.ServeHTTP(w, req)
	return w.Result(), w.Body.Bytes(), nil
}

func (h *ContactTestHelper) UpdateContactSubmission(id string, reqBody interface{}) (*http.Response, []byte, error) {
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPatch, "/contact/"+id, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if h.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+h.AuthToken)
	}
	w := httptest.NewRecorder()
	h.Router.ServeHTTP(w, req)
	return w.Result(), w.Body.Bytes(), nil
}
