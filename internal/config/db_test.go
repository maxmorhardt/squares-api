package config

import (
	"testing"

	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitDB_ConnectionError(t *testing.T) {
	cfg := &model.AppConfig{}
	cfg.DB.Host = "127.0.0.1"
	cfg.DB.Port = 1
	cfg.DB.User = "u"
	cfg.DB.Password = "p"
	cfg.DB.Name = "n"
	cfg.DB.SSLMode = "disable"

	db, err := InitDB(cfg)

	require.Error(t, err)
	assert.Nil(t, db)
}

func TestFormatDSN(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		user     string
		password string
		dbname   string
		sslmode  string
		expected string
	}{
		{
			name:     "standard connection",
			host:     "localhost",
			port:     5432,
			user:     "admin",
			password: "secret",
			dbname:   "mydb",
			sslmode:  "disable",
			expected: "host=localhost port=5432 user=admin password=secret dbname=mydb sslmode=disable",
		},
		{
			name:     "remote host with ssl",
			host:     "db.example.com",
			port:     5433,
			user:     "app",
			password: "p@ss!",
			dbname:   "production",
			sslmode:  "require",
			expected: "host=db.example.com port=5433 user=app password=p@ss! dbname=production sslmode=require",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDSN(tt.host, tt.port, tt.user, tt.password, tt.dbname, tt.sslmode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateReadReplicaConfig(t *testing.T) {
	complete := model.DatabaseConfig{
		ReadHost:     "replica",
		ReadPort:     5432,
		ReadUser:     "reader",
		ReadPassword: "secret",
		ReadName:     "mydb",
		ReadSSLMode:  "require",
	}

	tests := []struct {
		name      string
		mutate    func(c *model.DatabaseConfig)
		wantErr   bool
		wantInMsg string
	}{
		{name: "complete config", mutate: func(*model.DatabaseConfig) {}, wantErr: false},
		{name: "missing port", mutate: func(c *model.DatabaseConfig) { c.ReadPort = 0 }, wantErr: true, wantInMsg: "DB_READ_PORT"},
		{name: "missing user", mutate: func(c *model.DatabaseConfig) { c.ReadUser = "" }, wantErr: true, wantInMsg: "DB_READ_USER"},
		{name: "missing password", mutate: func(c *model.DatabaseConfig) { c.ReadPassword = "" }, wantErr: true, wantInMsg: "DB_READ_PASSWORD"},
		{name: "missing name", mutate: func(c *model.DatabaseConfig) { c.ReadName = "" }, wantErr: true, wantInMsg: "DB_READ_NAME"},
		{name: "missing ssl mode", mutate: func(c *model.DatabaseConfig) { c.ReadSSLMode = "" }, wantErr: true, wantInMsg: "DB_READ_SSL_MODE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbCfg := complete
			tt.mutate(&dbCfg)

			err := validateReadReplicaConfig(&model.AppConfig{DB: dbCfg})

			if !tt.wantErr {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantInMsg)
		})
	}
}
