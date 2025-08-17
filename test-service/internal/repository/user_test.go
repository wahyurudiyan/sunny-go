package repository

import (
	"context"
	"testing"

	"github.com/yourorg/test-service/internal/models"
	"github.com/yourorg/test-service/test/fixtures"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestUserRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	require.NoError(t, err)

	repo := NewUserRepository(gormDB)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		user := fixtures.NewCreateUserRequest()
		
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO "users"`).
			WithArgs(user.Username, user.Email, user.FirstName, user.LastName, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectCommit()

		createdUser, err := repo.Create(ctx, user)
		
		assert.NoError(t, err)
		assert.Equal(t, int64(1), createdUser.ID)
		assert.Equal(t, user.Username, createdUser.Username)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		user := fixtures.NewCreateUserRequest()
		
		mock.ExpectBegin()
		mock.ExpectQuery(`INSERT INTO "users"`).
			WithArgs(user.Username, user.Email, user.FirstName, user.LastName, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		_, err := repo.Create(ctx, user)
		
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	require.NoError(t, err)

	repo := NewUserRepository(gormDB)
	ctx := context.Background()

	t.Run("user found", func(t *testing.T) {
		expectedUser := fixtures.NewUser()
		expectedUser.ID = 1

		rows := sqlmock.NewRows([]string{"id", "username", "email", "first_name", "last_name", "created_at", "updated_at"}).
			AddRow(expectedUser.ID, expectedUser.Username, expectedUser.Email, expectedUser.FirstName, expectedUser.LastName,
				   expectedUser.CreatedAt, expectedUser.UpdatedAt)

		mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1`).
			WithArgs(1).
			WillReturnRows(rows)

		user, err := repo.GetByID(ctx, 1)
		
		assert.NoError(t, err)
		assert.Equal(t, expectedUser.ID, user.ID)
		assert.Equal(t, expectedUser.Username, user.Username)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("user not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1`).
			WithArgs(1).
			WillReturnError(gorm.ErrRecordNotFound)

		_, err := repo.GetByID(ctx, 1)
		
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
