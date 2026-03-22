package repository_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"backend-pretest-ai/internal/domain"
	"backend-pretest-ai/internal/repository"
)

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	assert.NoError(t, err)

	return gormDB, mock
}

func TestUserRepository_FindByEmail(t *testing.T) {
	gormDB, mock := setupTestDB(t)
	repo := repository.NewTestUserRepository(gormDB)

	email := "test@example.com"
	user := &domain.User{
		ID:    "user-123",
		Email: email,
		Name:  "Test User",
	}

	rows := sqlmock.NewRows([]string{"id", "email", "name"}).
		AddRow(user.ID, user.Email, user.Name)

	mock.ExpectQuery(`SELECT \* FROM "users" WHERE email = \$1 ORDER BY "users"\."id" LIMIT \$2`).
		WithArgs(email, 1).
		WillReturnRows(rows)

	result, err := repo.FindByEmail(context.Background(), email)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, email, result.Email)
}

func TestUserRepository_Create(t *testing.T) {
	gormDB, mock := setupTestDB(t)
	repo := repository.NewTestUserRepository(gormDB)

	user := &domain.User{
		Name:  "New User",
		Email: "new@example.com",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "users" (.+) VALUES (.+) RETURNING "id"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("new-id"))
	mock.ExpectCommit()

	err := repo.Create(context.Background(), user)

	assert.NoError(t, err)
}
