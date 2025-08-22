package model

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/goplus/builder/spx-backend/internal/model/modeltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAiResourceTableName(t *testing.T) {
	ar := AiResource{}
	assert.Equal(t, "aiResource", ar.TableName())
}

func TestAiResourceDBOperations(t *testing.T) {
	db, _, closeDB, err := modeltest.NewMockDB()
	require.NoError(t, err)
	closeDB()
	aiResourceDBColumns, err := modeltest.ExtractDBColumns(db, AiResource{})
	require.NoError(t, err)
	generateAiResourceDBRows, err := modeltest.NewDBRowsGenerator(db, AiResource{})
	require.NoError(t, err)

	testAiResource := AiResource{
		Model: Model{ID: 1},
		URL:   "https://example.com/ai-resource.json",
	}

	t.Run("Create", func(t *testing.T) {
		db, dbMock, closeDB, err := modeltest.NewMockDB()
		require.NoError(t, err)
		defer closeDB()

		dbMock.ExpectBegin()

		dbMockStmt := db.Session(&gorm.Session{DryRun: true, SkipDefaultTransaction: true}).
			Create(&testAiResource).
			Statement
		dbMockArgs := modeltest.ToDriverValueSlice(dbMockStmt.Vars...)
		dbMockArgs[1] = sqlmock.AnyArg() // CreatedAt
		dbMockArgs[2] = sqlmock.AnyArg() // UpdatedAt
		dbMock.ExpectExec(regexp.QuoteMeta(dbMockStmt.SQL.String())).
			WithArgs(dbMockArgs...).
			WillReturnResult(sqlmock.NewResult(1, 1))

		dbMock.ExpectCommit()

		err = db.WithContext(context.Background()).Create(&testAiResource).Error
		require.NoError(t, err)

		require.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("Find", func(t *testing.T) {
		db, dbMock, closeDB, err := modeltest.NewMockDB()
		require.NoError(t, err)
		defer closeDB()

		dbMockStmt := db.Session(&gorm.Session{DryRun: true}).
			Where("id = ?", testAiResource.ID).
			First(&AiResource{}).
			Statement
		dbMockArgs := modeltest.ToDriverValueSlice(dbMockStmt.Vars...)
		dbMock.ExpectQuery(regexp.QuoteMeta(dbMockStmt.SQL.String())).
			WithArgs(dbMockArgs...).
			WillReturnRows(sqlmock.NewRows(aiResourceDBColumns).AddRows(generateAiResourceDBRows(testAiResource)...))

		var result AiResource
		err = db.WithContext(context.Background()).Where("id = ?", testAiResource.ID).First(&result).Error
		require.NoError(t, err)
		assert.Equal(t, testAiResource.ID, result.ID)
		assert.Equal(t, testAiResource.URL, result.URL)

		require.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("Update", func(t *testing.T) {
		db, dbMock, closeDB, err := modeltest.NewMockDB()
		require.NoError(t, err)
		defer closeDB()

		updatedResource := testAiResource
		updatedResource.URL = "https://example.com/updated-resource.json"

		dbMock.ExpectBegin()

		dbMockStmt := db.Session(&gorm.Session{DryRun: true, SkipDefaultTransaction: true}).
			Model(&testAiResource).
			Updates(map[string]interface{}{"url": updatedResource.URL}).
			Statement
		dbMockArgs := modeltest.ToDriverValueSlice(dbMockStmt.Vars...)
		dbMockArgs[0] = sqlmock.AnyArg() // UpdatedAt
		dbMock.ExpectExec(regexp.QuoteMeta(dbMockStmt.SQL.String())).
			WithArgs(dbMockArgs...).
			WillReturnResult(sqlmock.NewResult(0, 1))

		dbMock.ExpectCommit()

		err = db.WithContext(context.Background()).Model(&testAiResource).Updates(map[string]interface{}{"url": updatedResource.URL}).Error
		require.NoError(t, err)

		require.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("Delete", func(t *testing.T) {
		db, dbMock, closeDB, err := modeltest.NewMockDB()
		require.NoError(t, err)
		defer closeDB()

		dbMock.ExpectBegin()

		dbMockStmt := db.Session(&gorm.Session{DryRun: true, SkipDefaultTransaction: true}).
			Delete(&testAiResource).
			Statement
		dbMockArgs := modeltest.ToDriverValueSlice(dbMockStmt.Vars...)
		dbMockArgs[0] = sqlmock.AnyArg() // DeletedAt
		dbMock.ExpectExec(regexp.QuoteMeta(dbMockStmt.SQL.String())).
			WithArgs(dbMockArgs...).
			WillReturnResult(sqlmock.NewResult(0, 1))

		dbMock.ExpectCommit()

		err = db.WithContext(context.Background()).Delete(&testAiResource).Error
		require.NoError(t, err)

		require.NoError(t, dbMock.ExpectationsWereMet())
	})
}

func TestAiResourceValidation(t *testing.T) {
	t.Run("ValidURL", func(t *testing.T) {
		ar := AiResource{
			URL: "https://example.com/resource.json",
		}
		assert.NotEmpty(t, ar.URL)
		assert.Contains(t, ar.URL, "https://")
	})

	t.Run("EmptyURL", func(t *testing.T) {
		ar := AiResource{
			URL: "",
		}
		assert.Empty(t, ar.URL)
	})

	t.Run("RelativeURL", func(t *testing.T) {
		ar := AiResource{
			URL: "/api/resources/123",
		}
		assert.NotEmpty(t, ar.URL)
		assert.True(t, len(ar.URL) > 0)
	})
}