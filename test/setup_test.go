package test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/routes"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	postgresdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	postgresTag      = "postgres:18-alpine"
	postgresDBName   = "squares"
	postgresUser     = "test_user"
	postgresPassword = "test_password"
)

var (
	router            *gin.Engine
	postgresContainer *postgres.PostgresContainer
	contestService    service.ContestService
	contactService    service.ContactService
	oidcUser          string
	authToken         string
)

func TestMain(m *testing.M) {
	// setup
	_ = godotenv.Load(".env")
	ctx := context.Background()
	gin.SetMode(gin.TestMode)

	setupTestDatabase(ctx)
	setupMockServices()
	setupAuth()
	router = setupTestRouter()

	// run tests
	code := m.Run()

	// teardown
	teardownTestDatabase(ctx)
	os.Exit(code)
}

func setupTestDatabase(ctx context.Context) {
	// start a postgres container
	var err error
	postgresContainer, err = postgres.Run(ctx,
		postgresTag,
		postgres.WithDatabase(postgresDBName),
		postgres.WithUsername(postgresUser),
		postgres.WithPassword(postgresPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)

	if err != nil {
		slog.Error("failed to start postgres container", "error", err)
		os.Exit(1)
	}

	// get connection string from container
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		slog.Error("failed to get connection string", "error", err)
		os.Exit(1)
	}

	// connect to the test database
	config.DB, err = gorm.Open(postgresdriver.Open(connStr), &gorm.Config{})
	if err != nil {
		slog.Error("failed to connect to test database", "error", err)
		os.Exit(1)
	}

	// run migrations
	models := []any{
		&model.Contest{},
		&model.Square{},
		&model.QuarterResult{},
		&model.ContactSubmission{},
	}
	for _, model := range models {
		if err := config.DB.AutoMigrate(model); err != nil {
			slog.Error("failed to migrate model", "error", err)
			os.Exit(1)
		}
	}
}

func setupMockServices() {
	config.InitOIDC()
}

func setupAuth() {
	// get credentials from environment
	authentikURL := "https://login.maxstash.io"
	clientID := os.Getenv("OIDC_CLIENT_ID")
	oidcUser = os.Getenv("OIDC_TEST_USERNAME")
	password := os.Getenv("OIDC_TEST_PASSWORD")

	if clientID == "" || oidcUser == "" || password == "" {
		slog.Error("OIDC environment variables missing")
		panic("OIDC environment variables must be set")
	}

	// request token using client credentials grant
	tokenURL := fmt.Sprintf("%s/application/o/token/", authentikURL)
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", clientID)
	data.Set("scope", "openid email profile")
	data.Set("username", oidcUser)
	data.Set("password", password)

	requestBody := data.Encode()
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(requestBody))
	if err != nil {
		slog.Error("failed to create token request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("failed to request token", "error", err)
		return
	}
	
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("failed to close response body", "error", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || err != nil {
		slog.Error("failed to get auth token", "status", resp.StatusCode, "body", string(body))
		return
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}

	json.Unmarshal(body, &tokenResp)
	authToken = tokenResp.AccessToken
	if authToken == "" {
		slog.Error("no access token in response")
		return
	}

	slog.Info("successfully obtained auth token")
}

func setupTestRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	contestRepo := repository.NewContestRepository()
	contactRepo := repository.NewContactRepository()

	authService := service.NewAuthService()
	redisService := service.NewRedisService()
	contestService = service.NewContestService(contestRepo, redisService, authService)
	contactService = service.NewContactService(contactRepo)
	wsService := service.NewWebSocketService()

	contestHandler := handler.NewContestHandler(contestService, authService)
	contactHandler := handler.NewContactHandler(contactService)
	wsHandler := handler.NewWebSocketHandler(wsService, contestRepo)

	routes.RegisterRootRoutes(r.Group(""), contactHandler)
	routes.RegisterContestRoutes(r.Group("/contests"), contestHandler)
	routes.RegisterWebSocketRoutes(r.Group("/ws"), wsHandler)

	return r
}

func teardownTestDatabase(ctx context.Context) {
	if postgresContainer != nil {
		if err := postgresContainer.Terminate(ctx); err != nil {
			slog.Warn("failed to terminate postgres container", "error", err)
		}
	}
}
