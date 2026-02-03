package test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
	"github.com/maxmorhardt/squares-api/internal/bootstrap"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	postgresTag      = "postgres:17-alpine"
	postgresDBName   = "squares"
	postgresUser     = "test_user"
	postgresPassword = "test_password"
	redisTag         = "redis:8-alpine"
)

var (
	router            *gin.Engine
	postgresContainer *postgres.PostgresContainer
	redisContainer    *redis.RedisContainer
	oidcUser          string
	authToken         string
)

func TestMain(m *testing.M) {
	_ = godotenv.Load("../.env.test")
	ctx := context.Background()
	gin.SetMode(gin.TestMode)

	setupPostgresContainer(ctx)
	setupRedisContainer(ctx)
	setupAuth()
	router = bootstrap.NewServer()

	code := m.Run()

	teardownContainers(ctx)
	os.Exit(code)
}

func setupPostgresContainer(ctx context.Context) {
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

	// container connection details
	host, err := postgresContainer.Host(ctx)
	if err != nil {
		slog.Error("failed to get container host", "error", err)
		os.Exit(1)
	}

	port, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		slog.Error("failed to get container port", "error", err)
		os.Exit(1)
	}

	_ = os.Setenv("DB_HOST", host)
	_ = os.Setenv("DB_PORT", port.Port())
	_ = os.Setenv("DB_USER", postgresUser)
	_ = os.Setenv("DB_PASSWORD", postgresPassword)
	_ = os.Setenv("DB_NAME", postgresDBName)
	_ = os.Setenv("DB_SSL_MODE", "disable")

	slog.Info("postgres container configured", "host", host, "port", port.Port())
}

func setupRedisContainer(ctx context.Context) {
	var err error
	redisContainer, err = redis.Run(ctx,
		redisTag,
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").WithStartupTimeout(30*time.Second)),
	)

	if err != nil {
		slog.Error("failed to start redis container", "error", err)
		os.Exit(1)
	}

	// container connection details
	host, err := redisContainer.Host(ctx)
	if err != nil {
		slog.Error("failed to get redis container host", "error", err)
		os.Exit(1)
	}

	port, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		slog.Error("failed to get redis container port", "error", err)
		os.Exit(1)
	}

	redisHost := host + ":" + port.Port()
	_ = os.Setenv("REDIS_HOST", redisHost)
	_ = os.Setenv("REDIS_PASSWORD", "")

	slog.Info("redis container configured", "host", redisHost)
}

func setupAuth() {
	// get credentials from environment
	authUrl := "https://login.maxstash.io/application/o/token/"
	clientID := os.Getenv("OIDC_CLIENT_ID")
	oidcUser = os.Getenv("OIDC_USER")
	password := os.Getenv("OIDC_PASSWORD")

	if clientID == "" || oidcUser == "" || password == "" {
		slog.Error("OIDC environment variables missing")
		panic("OIDC environment variables must be set")
	}

	// request token using client credentials grant
	client := resty.New().SetTimeout(10 * time.Second)

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}

	resp, err := client.R().
		SetFormData(map[string]string{
			"grant_type": "client_credentials",
			"client_id":  clientID,
			"username":   oidcUser,
			"password":   password,
			"scope":      "openid email profile",
		}).
		SetResult(&tokenResp).
		Post(authUrl)

	if err != nil {
		slog.Error("failed to request token", "error", err)
		return
	}

	if !resp.IsSuccess() {
		slog.Error("failed to get auth token", "status", resp.StatusCode(), "body", resp.String())
		return
	}

	authToken = tokenResp.AccessToken
	if authToken == "" {
		slog.Error("no access token in response")
		return
	}

	slog.Info("successfully obtained auth token")
}

func teardownContainers(ctx context.Context) {
	if postgresContainer != nil {
		if err := postgresContainer.Terminate(ctx); err != nil {
			slog.Warn("failed to terminate postgres container", "error", err)
		}
	}

	if redisContainer != nil {
		if err := redisContainer.Terminate(ctx); err != nil {
			slog.Warn("failed to terminate redis container", "error", err)
		}
	}
}
