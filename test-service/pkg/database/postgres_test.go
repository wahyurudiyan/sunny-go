package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnect(t *testing.T) {
	tests := []struct {
		name      string
		dbURL     string
		wantError bool
	}{
		{
			name:      "invalid connection string",
			dbURL:     "invalid://connection",
			wantError: true,
		},
		{
			name:      "sqlite in-memory",
			dbURL:     "sqlite://file::memory:?cache=shared",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := Connect(tt.dbURL)
			
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, db)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, db)
				
				// Clean up
				if db != nil {
					sqlDB, _ := db.DB()
					if sqlDB != nil {
						sqlDB.Close()
					}
				}
			}
		})
	}
}
