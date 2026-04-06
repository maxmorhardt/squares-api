package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
