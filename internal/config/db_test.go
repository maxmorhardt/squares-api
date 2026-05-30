package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestDB_ReturnsNilWhenNotInitialized(t *testing.T) {
	original := database
	database = nil
	defer func() { database = original }()

	assert.Nil(t, DB())
}

func TestValidateReadReplicaConfig(t *testing.T) {
	complete := databaseConfig{
		ReadHost:     "replica",
		ReadPort:     5432,
		ReadUser:     "reader",
		ReadPassword: "secret",
		ReadName:     "mydb",
		ReadSSLMode:  "require",
	}

	tests := []struct {
		name      string
		mutate    func(c *databaseConfig)
		wantPanic bool
		wantInMsg string
	}{
		{name: "complete config", mutate: func(*databaseConfig) {}, wantPanic: false},
		{name: "missing port", mutate: func(c *databaseConfig) { c.ReadPort = 0 }, wantPanic: true, wantInMsg: "DB_READ_PORT"},
		{name: "missing user", mutate: func(c *databaseConfig) { c.ReadUser = "" }, wantPanic: true, wantInMsg: "DB_READ_USER"},
		{name: "missing password", mutate: func(c *databaseConfig) { c.ReadPassword = "" }, wantPanic: true, wantInMsg: "DB_READ_PASSWORD"},
		{name: "missing name", mutate: func(c *databaseConfig) { c.ReadName = "" }, wantPanic: true, wantInMsg: "DB_READ_NAME"},
		{name: "missing ssl mode", mutate: func(c *databaseConfig) { c.ReadSSLMode = "" }, wantPanic: true, wantInMsg: "DB_READ_SSL_MODE"},
	}

	original := cfg
	defer func() { cfg = original }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbCfg := complete
			tt.mutate(&dbCfg)
			cfg = &config{DB: dbCfg}

			if !tt.wantPanic {
				assert.NotPanics(t, validateReadReplicaConfig)
				return
			}

			defer func() {
				r := recover()
				require.NotNil(t, r, "expected panic")
				assert.Contains(t, r.(string), tt.wantInMsg)
			}()
			validateReadReplicaConfig()
		})
	}
}
