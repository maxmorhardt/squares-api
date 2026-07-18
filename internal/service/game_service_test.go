package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/mocks"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGameService_GetUpcoming(t *testing.T) {
	g := mocks.NewGameRepository(t)
	g.EXPECT().GetUpcoming(mock.Anything).Return([]model.Game{{ESPNID: "1"}, {ESPNID: "2"}}, nil)

	got, err := gameSvc(g, mocks.NewContestRepository(t)).GetUpcoming(context.Background())
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func gameSvc(gameRepo *mocks.GameRepository, contestRepo *mocks.ContestRepository) service.GameService {
	return service.NewGameService(gameRepo, contestRepo, anyNats())
}

func TestGameService_GetUpcoming_DBError(t *testing.T) {
	g := mocks.NewGameRepository(t)
	g.EXPECT().GetUpcoming(mock.Anything).Return(nil, errors.New("boom"))

	_, err := gameSvc(g, mocks.NewContestRepository(t)).GetUpcoming(context.Background())
	assert.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestGameService_Ingest_UpsertsAndSkipsScheduled(t *testing.T) {
	g := mocks.NewGameRepository(t)
	// scheduled game: row is refreshed but nothing is scored or reconciled
	g.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil).Once()

	games := []model.ESPNGame{{ESPNID: "1", State: "pre"}}
	newScores, err := gameSvc(g, mocks.NewContestRepository(t)).Ingest(context.Background(), games)
	require.NoError(t, err)
	assert.Zero(t, newScores)
}

func TestGameService_Ingest_RecordsScoresAndSyncs(t *testing.T) {
	g := mocks.NewGameRepository(t)
	g.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil).Once()
	// in Q2, so Q1 is complete and gets recorded
	g.EXPECT().UpsertScore(mock.Anything, mock.Anything).Return(true, nil).Once()
	// live game triggers a sync of linked contests
	g.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Game{Status: model.GameStatusInProgress}, nil).Once()

	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByGameID(mock.Anything, mock.Anything).Return([]model.Contest{}, nil).Once()

	games := []model.ESPNGame{{ESPNID: "1", State: "in", Period: 2, HomeLine: []int{7}, AwayLine: []int{3}}}
	newScores, err := gameSvc(g, c).Ingest(context.Background(), games)
	require.NoError(t, err)
	assert.Equal(t, 1, newScores)
}

func TestGameService_Ingest_UpsertErrorSkipsGame(t *testing.T) {
	g := mocks.NewGameRepository(t)
	g.EXPECT().Upsert(mock.Anything, mock.Anything).Return(errors.New("db")).Once()
	// UpsertScore / SyncGame must not run when the game upsert fails

	games := []model.ESPNGame{{ESPNID: "1", State: "in"}}
	newScores, err := gameSvc(g, mocks.NewContestRepository(t)).Ingest(context.Background(), games)
	require.NoError(t, err)
	assert.Zero(t, newScores)
}

func TestGameService_Activity(t *testing.T) {
	kickoff := time.Now().Add(2 * time.Hour)
	g := mocks.NewGameRepository(t)
	g.EXPECT().HasLiveGame(mock.Anything).Return(true, nil)
	g.EXPECT().NextKickoff(mock.Anything).Return(kickoff, nil)

	act, err := gameSvc(g, mocks.NewContestRepository(t)).Activity(context.Background())
	require.NoError(t, err)
	assert.True(t, act.Live)
	assert.Equal(t, kickoff, act.NextKickoff)
}

func TestGameService_Activity_LiveError(t *testing.T) {
	g := mocks.NewGameRepository(t)
	g.EXPECT().HasLiveGame(mock.Anything).Return(false, errors.New("db"))

	_, err := gameSvc(g, mocks.NewContestRepository(t)).Activity(context.Background())
	assert.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestGameService_Activity_KickoffError(t *testing.T) {
	g := mocks.NewGameRepository(t)
	g.EXPECT().HasLiveGame(mock.Anything).Return(false, nil)
	g.EXPECT().NextKickoff(mock.Anything).Return(time.Time{}, errors.New("db"))

	_, err := gameSvc(g, mocks.NewContestRepository(t)).Activity(context.Background())
	assert.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestGameService_SyncGame_AdvancesStartedContest(t *testing.T) {
	gameID := uuid.New()
	g := mocks.NewGameRepository(t)
	g.EXPECT().GetByID(mock.Anything, gameID).Return(liveGame(gameID, model.GameScore{Quarter: 1, HomeScore: 7, AwayScore: 3}), nil)

	contest := startedContest(model.ContestStatusQ1, &model.Game{ID: gameID})
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByGameID(mock.Anything, gameID).Return([]model.Contest{contest}, nil)
	c.EXPECT().Update(mock.Anything, mock.MatchedBy(func(ct *model.Contest) bool {
		return ct.Status == model.ContestStatusQ2
	})).Return(nil).Once()

	require.NoError(t, gameSvc(g, c).SyncGame(context.Background(), gameID))
}

func liveGame(gameID uuid.UUID, scores ...model.GameScore) *model.Game {
	return &model.Game{ID: gameID, Status: model.GameStatusInProgress, Scores: scores}
}

func startedContest(status model.ContestStatus, game *model.Game) model.Contest {
	labels, _ := json.Marshal([]int8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
	squares := make([]model.Square, 0, 100)
	for r := 0; r < 10; r++ {
		for c := 0; c < 10; c++ {
			squares = append(squares, model.Square{Row: r, Col: c, Owner: "u", OwnerName: "U"})
		}
	}
	return model.Contest{ID: uuid.New(), Status: status, Game: game, XLabels: labels, YLabels: labels, Squares: squares}
}

func TestGameService_SyncGame_AutoStartsAndBackfills(t *testing.T) {
	gameID := uuid.New()
	g := mocks.NewGameRepository(t)
	g.EXPECT().GetByID(mock.Anything, gameID).Return(liveGame(gameID,
		model.GameScore{Quarter: 1, HomeScore: 7, AwayScore: 3},
		model.GameScore{Quarter: 2, HomeScore: 14, AwayScore: 10},
	), nil)

	contest := startedContest(model.ContestStatusActive, &model.Game{ID: gameID})
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByGameID(mock.Anything, gameID).Return([]model.Contest{contest}, nil)

	// auto-start (Q1) then backfill Q1 (->Q2) and Q2 (->Q3)
	var lastStatus model.ContestStatus
	c.EXPECT().Update(mock.Anything, mock.Anything).Run(func(_ context.Context, ct *model.Contest) {
		lastStatus = ct.Status
	}).Return(nil)

	require.NoError(t, gameSvc(g, c).SyncGame(context.Background(), gameID))
	assert.Equal(t, model.ContestStatusQ3, lastStatus)
}

func TestGameService_SyncGame_SkipsWhenGameNotLive(t *testing.T) {
	gameID := uuid.New()
	g := mocks.NewGameRepository(t)
	g.EXPECT().GetByID(mock.Anything, gameID).Return(&model.Game{ID: gameID, Status: model.GameStatusScheduled}, nil)

	contest := startedContest(model.ContestStatusActive, &model.Game{ID: gameID})
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByGameID(mock.Anything, gameID).Return([]model.Contest{contest}, nil)
	// no Update expected: game hasn't started

	require.NoError(t, gameSvc(g, c).SyncGame(context.Background(), gameID))
}

func TestGameService_SyncGame_SkipsWhenGridNotFull(t *testing.T) {
	gameID := uuid.New()
	g := mocks.NewGameRepository(t)
	g.EXPECT().GetByID(mock.Anything, gameID).Return(liveGame(gameID, model.GameScore{Quarter: 1, HomeScore: 7, AwayScore: 3}), nil)

	contest := startedContest(model.ContestStatusActive, &model.Game{ID: gameID})
	contest.Squares[0].Owner = "" // leave one square unclaimed
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByGameID(mock.Anything, gameID).Return([]model.Contest{contest}, nil)
	// no Update expected: grid not ready to start

	require.NoError(t, gameSvc(g, c).SyncGame(context.Background(), gameID))
}

func TestGameService_SyncGame_SkipsAlreadyAppliedQuarter(t *testing.T) {
	gameID := uuid.New()
	g := mocks.NewGameRepository(t)
	g.EXPECT().GetByID(mock.Anything, gameID).Return(liveGame(gameID, model.GameScore{Quarter: 1, HomeScore: 7, AwayScore: 3}), nil)

	// contest already past Q1; the Q1 score must not re-advance it
	contest := startedContest(model.ContestStatusQ2, &model.Game{ID: gameID})
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByGameID(mock.Anything, gameID).Return([]model.Contest{contest}, nil)
	// no Update expected

	require.NoError(t, gameSvc(g, c).SyncGame(context.Background(), gameID))
}

func finalGame(gameID uuid.UUID, scores ...model.GameScore) *model.Game {
	return &model.Game{ID: gameID, Status: model.GameStatusFinal, Scores: scores}
}

func TestGameService_SyncGame_FinalizesWhenGameEnds(t *testing.T) {
	gameID := uuid.New()
	g := mocks.NewGameRepository(t)
	g.EXPECT().GetByID(mock.Anything, gameID).Return(finalGame(gameID,
		model.GameScore{Quarter: 1, HomeScore: 7, AwayScore: 3},
		model.GameScore{Quarter: 4, HomeScore: 21, AwayScore: 17},
	), nil)

	// still ACTIVE (never auto-started) when the game ends
	contest := startedContest(model.ContestStatusActive, &model.Game{ID: gameID})
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByGameID(mock.Anything, gameID).Return([]model.Contest{contest}, nil)

	// exactly one Update: labels assigned and status jumps straight to FINISHED
	var finalStatus model.ContestStatus
	c.EXPECT().Update(mock.Anything, mock.Anything).Run(func(_ context.Context, ct *model.Contest) {
		finalStatus = ct.Status
	}).Return(nil).Once()

	require.NoError(t, gameSvc(g, c).SyncGame(context.Background(), gameID))
	assert.Equal(t, model.ContestStatusFinished, finalStatus)
}

func TestGameService_SyncGame_FinalizeUpdateError(t *testing.T) {
	gameID := uuid.New()
	g := mocks.NewGameRepository(t)
	g.EXPECT().GetByID(mock.Anything, gameID).Return(finalGame(gameID), nil)

	contest := startedContest(model.ContestStatusActive, &model.Game{ID: gameID})
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByGameID(mock.Anything, gameID).Return([]model.Contest{contest}, nil)
	c.EXPECT().Update(mock.Anything, mock.Anything).Return(errors.New("db")).Once()

	// SyncGame logs and swallows the reconcile error, so it still returns nil
	require.NoError(t, gameSvc(g, c).SyncGame(context.Background(), gameID))
}

func TestGameService_SyncGame_FinalizesUnfilledGrid(t *testing.T) {
	gameID := uuid.New()
	g := mocks.NewGameRepository(t)
	g.EXPECT().GetByID(mock.Anything, gameID).Return(finalGame(gameID,
		model.GameScore{Quarter: 1, HomeScore: 7, AwayScore: 3},
	), nil)

	// grid never filled, but the game is over: still resolve to FINISHED
	contest := startedContest(model.ContestStatusActive, &model.Game{ID: gameID})
	contest.Squares[0].Owner = ""
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByGameID(mock.Anything, gameID).Return([]model.Contest{contest}, nil)

	var finalStatus model.ContestStatus
	c.EXPECT().Update(mock.Anything, mock.Anything).Run(func(_ context.Context, ct *model.Contest) {
		finalStatus = ct.Status
	}).Return(nil).Once()

	require.NoError(t, gameSvc(g, c).SyncGame(context.Background(), gameID))
	assert.Equal(t, model.ContestStatusFinished, finalStatus)
}
