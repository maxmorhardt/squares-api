package test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/bootstrap"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	postgresTag      = "postgres:17-alpine"
	postgresDBName   = "squares"
	postgresUser     = "test_user"
	postgresPassword = "test_password"
	natsTag          = "nats:2.10-alpine"

	// fixed identities used across all integration tests
	ownerUser   = "owner"
	memberUser  = "member"
	ownerToken  = ownerUser
	memberToken = memberUser
)

var (
	router            *gin.Engine
	postgresContainer *postgres.PostgresContainer
	natsContainer     testcontainers.Container
)

type fakeTokenVerifier struct{}

func (fakeTokenVerifier) Verify(_ context.Context, token string) (*model.Claims, error) {
	return &model.Claims{Username: token, Name: token}, nil
}

func TestMain(m *testing.M) {
	ctx := context.Background()
	gin.SetMode(gin.TestMode)

	setupPostgresContainer(ctx)
	setupNatsContainer(ctx)

	_ = os.Setenv("SMTP_HOST", "localhost")
	_ = os.Setenv("SMTP_PORT", "587")
	_ = os.Setenv("SMTP_USER", "test@example.com")
	_ = os.Setenv("SMTP_PASSWORD", "test")
	_ = os.Setenv("SUPPORT_EMAIL", "support@example.com")
	_ = os.Setenv("OIDC_CLIENT_ID", "test-client")
	_ = os.Setenv("TURNSTILE_SECRET_KEY", "test-key")

	deps := buildDeps()
	router = bootstrap.NewServer(deps)

	code := m.Run()

	teardownContainers(ctx)
	os.Exit(code)
}

func setupPostgresContainer(ctx context.Context) {
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

func setupNatsContainer(ctx context.Context) {
	var err error
	natsContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        natsTag,
			ExposedPorts: []string{"4222/tcp"},
			WaitingFor:   wait.ForLog("Server is ready").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		slog.Error("failed to start NATS container", "error", err)
		os.Exit(1)
	}

	host, err := natsContainer.Host(ctx)
	if err != nil {
		slog.Error("failed to get NATS container host", "error", err)
		os.Exit(1)
	}

	port, err := natsContainer.MappedPort(ctx, "4222")
	if err != nil {
		slog.Error("failed to get NATS container port", "error", err)
		os.Exit(1)
	}

	natsURL := "nats://" + host + ":" + port.Port()
	_ = os.Setenv("NATS_URL", natsURL)

	slog.Info("NATS container configured", "url", natsURL)
}

func teardownContainers(ctx context.Context) {
	if postgresContainer != nil {
		if err := postgresContainer.Terminate(ctx); err != nil {
			slog.Warn("failed to terminate postgres container", "error", err)
		}
	}

	if natsContainer != nil {
		if err := natsContainer.Terminate(ctx); err != nil {
			slog.Warn("failed to terminate NATS container", "error", err)
		}
	}
}

func buildDeps() *bootstrap.Dependencies {
	cfg, err := config.LoadEnv()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := config.InitDB(cfg)
	if err != nil {
		slog.Error("failed to init db", "error", err)
		os.Exit(1)
	}

	nc, err := config.InitNATS(cfg)
	if err != nil {
		slog.Error("failed to init nats", "error", err)
		os.Exit(1)
	}

	return &bootstrap.Dependencies{
		Config:   cfg,
		DB:       db,
		NATS:     nc,
		Verifier: fakeTokenVerifier{},
	}
}
