package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGameRepository_Upsert_Insert(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewGameRepository(gdb)

	// existing lookup returns no rows -> insert path
	mock.ExpectQuery(`SELECT .* FROM "games"`).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "games"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.Upsert(context.Background(), &model.Game{ESPNID: "401", GameTime: time.Now()})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGameRepository_Upsert_Update(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewGameRepository(gdb)

	existingID := uuid.New()
	mock.ExpectQuery(`SELECT .* FROM "games"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "espn_id"}).AddRow(existingID, "401"))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "games"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	game := &model.Game{ESPNID: "401", HomeScore: 7, GameTime: time.Now()}
	err := repo.Upsert(context.Background(), game)
	require.NoError(t, err)
	assert.Equal(t, existingID, game.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGameRepository_GetByID(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewGameRepository(gdb)

	id := uuid.New()
	mock.ExpectQuery(`SELECT \* FROM "games"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "espn_id"}).AddRow(id, "401"))
	mock.ExpectQuery(`SELECT \* FROM "game_scores"`).
		WillReturnRows(sqlmock.NewRows([]string{"game_id", "quarter"}).AddRow(id, 1))

	game, err := repo.GetByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "401", game.ESPNID)
	assert.Len(t, game.Scores, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGameRepository_GetUpcoming(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewGameRepository(gdb)

	// first query anchors the window to the next scheduled game
	mock.ExpectQuery(`SELECT "game_time" FROM "games"`).
		WillReturnRows(sqlmock.NewRows([]string{"game_time"}).AddRow(time.Now().Add(24 * time.Hour)))
	// second query fetches games within the window
	mock.ExpectQuery(`SELECT \* FROM "games"`).
		WillReturnRows(sqlmock.NewRows([]string{"espn_id"}).AddRow("1").AddRow("2"))

	games, err := repo.GetUpcoming(context.Background())
	require.NoError(t, err)
	assert.Len(t, games, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGameRepository_GetUpcoming_NoScheduledGames(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewGameRepository(gdb)

	// no upcoming scheduled game: returns empty without a second query
	mock.ExpectQuery(`SELECT "game_time" FROM "games"`).
		WillReturnRows(sqlmock.NewRows([]string{"game_time"}))

	games, err := repo.GetUpcoming(context.Background())
	require.NoError(t, err)
	assert.Empty(t, games)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGameRepository_HasLiveGame(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewGameRepository(gdb)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "games"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	live, err := repo.HasLiveGame(context.Background())
	require.NoError(t, err)
	assert.True(t, live)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGameRepository_HasLiveGame_None(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewGameRepository(gdb)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "games"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	live, err := repo.HasLiveGame(context.Background())
	require.NoError(t, err)
	assert.False(t, live)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGameRepository_NextKickoff(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewGameRepository(gdb)

	want := time.Now().Add(3 * time.Hour)
	mock.ExpectQuery(`SELECT "game_time" FROM "games"`).
		WillReturnRows(sqlmock.NewRows([]string{"game_time"}).AddRow(want))

	got, err := repo.NextKickoff(context.Background())
	require.NoError(t, err)
	assert.WithinDuration(t, want, got, time.Second)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGameRepository_NextKickoff_None(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewGameRepository(gdb)

	// no upcoming scheduled game returns the zero time without error
	mock.ExpectQuery(`SELECT "game_time" FROM "games"`).
		WillReturnRows(sqlmock.NewRows([]string{"game_time"}))

	got, err := repo.NextKickoff(context.Background())
	require.NoError(t, err)
	assert.True(t, got.IsZero())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGameRepository_UpsertScore_Created(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewGameRepository(gdb)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "game_scores"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	created, err := repo.UpsertScore(context.Background(), &model.GameScore{GameID: uuid.New(), Quarter: 1, HomeScore: 7})
	require.NoError(t, err)
	assert.True(t, created)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGameRepository_UpsertScore_Duplicate(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewGameRepository(gdb)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "game_scores"`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	created, err := repo.UpsertScore(context.Background(), &model.GameScore{GameID: uuid.New(), Quarter: 1})
	require.NoError(t, err)
	assert.False(t, created)
	assert.NoError(t, mock.ExpectationsWereMet())
}
