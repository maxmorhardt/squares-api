package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
)

func CreateContest(router http.Handler, authToken, oidcUser, name, homeTeam, awayTeam string) (*model.Contest, int) {
	reqBody := model.CreateContestRequest{
		Owner:    oidcUser,
		Name:     name,
		HomeTeam: homeTeam,
		AwayTeam: awayTeam,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPut, "/contests", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var contest model.Contest
	_ = json.Unmarshal(w.Body.Bytes(), &contest)

	return &contest, w.Code
}

func GetContestByID(router http.Handler, contestID uuid.UUID) (*model.Contest, int) {
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/contests/%s", contestID), nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var contest model.Contest
	_ = json.Unmarshal(w.Body.Bytes(), &contest)

	return &contest, w.Code
}

func GetContestsByUser(router http.Handler, oidcUser, authToken string) (model.PaginatedContestResponse, int) {
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/contests/user/%s?page=1&limit=10", oidcUser), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response model.PaginatedContestResponse
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	return response, w.Code
}

func UpdateContest(router http.Handler, contestID uuid.UUID, authToken string, updateReq model.UpdateContestRequest) (model.Contest, int)  {
	body, _ := json.Marshal(updateReq)

	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s", contestID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var contest model.Contest
	_ = json.Unmarshal(w.Body.Bytes(), &contest)

	return contest, w.Code
}

func UpdateSquare(router http.Handler, contestID uuid.UUID, squareID uuid.UUID, authToken string, updateReq model.UpdateSquareRequest) int {
	body, _ := json.Marshal(updateReq)

	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/squares/%s", contestID, squareID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w.Code
}

func StartContest(router http.Handler, contestID uuid.UUID, authToken string) (*model.Contest, int)  {
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/start", contestID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var contest model.Contest
	_ = json.Unmarshal(w.Body.Bytes(), &contest)

	return &contest, w.Code
}

func SubmitQuarterResult(router http.Handler, contestID uuid.UUID, authToken string, reqBody model.QuarterResultRequest) int {
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/quarter-result", contestID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w.Code
}

func DeleteContest(router http.Handler, contestID uuid.UUID, authToken string) int {
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s", contestID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	return w.Code
}
