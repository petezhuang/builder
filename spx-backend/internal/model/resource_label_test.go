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

func TestResourceLabelTableName(t *testing.T) {
	rl := ResourceLabel{}
	assert.Equal(t, "resource_label", rl.TableName())
}

func TestResourceLabelDBOperations(t *testing.T) {
	db, _, closeDB, err := modeltest.NewMockDB()
	require.NoError(t, err)
	closeDB()
	resourceLabelDBColumns, err := modeltest.ExtractDBColumns(db, ResourceLabel{})
	require.NoError(t, err)
	generateResourceLabelDBRows, err := modeltest.NewDBRowsGenerator(db, ResourceLabel{})
	require.NoError(t, err)

	testResourceLabel := ResourceLabel{
		Model:        Model{ID: 1},
		AiResourceID: 123,
		LabelID:      456,
	}

	t.Run("Create", func(t *testing.T) {
		db, dbMock, closeDB, err := modeltest.NewMockDB()
		require.NoError(t, err)
		defer closeDB()

		dbMock.ExpectBegin()

		dbMockStmt := db.Session(&gorm.Session{DryRun: true, SkipDefaultTransaction: true}).
			Create(&testResourceLabel).
			Statement
		dbMockArgs := modeltest.ToDriverValueSlice(dbMockStmt.Vars...)
		dbMockArgs[1] = sqlmock.AnyArg() // CreatedAt
		dbMockArgs[2] = sqlmock.AnyArg() // UpdatedAt
		dbMock.ExpectExec(regexp.QuoteMeta(dbMockStmt.SQL.String())).
			WithArgs(dbMockArgs...).
			WillReturnResult(sqlmock.NewResult(1, 1))

		dbMock.ExpectCommit()

		err = db.WithContext(context.Background()).Create(&testResourceLabel).Error
		require.NoError(t, err)

		require.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("FindByAiResourceID", func(t *testing.T) {
		db, dbMock, closeDB, err := modeltest.NewMockDB()
		require.NoError(t, err)
		defer closeDB()

		dbMockStmt := db.Session(&gorm.Session{DryRun: true}).
			Where("aiResourceId = ?", testResourceLabel.AiResourceID).
			Find(&[]ResourceLabel{}).
			Statement
		dbMockArgs := modeltest.ToDriverValueSlice(dbMockStmt.Vars...)
		dbMock.ExpectQuery(regexp.QuoteMeta(dbMockStmt.SQL.String())).
			WithArgs(dbMockArgs...).
			WillReturnRows(sqlmock.NewRows(resourceLabelDBColumns).AddRows(generateResourceLabelDBRows(testResourceLabel)...))

		var results []ResourceLabel
		err = db.WithContext(context.Background()).Where("aiResourceId = ?", testResourceLabel.AiResourceID).Find(&results).Error
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, testResourceLabel.AiResourceID, results[0].AiResourceID)
		assert.Equal(t, testResourceLabel.LabelID, results[0].LabelID)

		require.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("FindByLabelID", func(t *testing.T) {
		db, dbMock, closeDB, err := modeltest.NewMockDB()
		require.NoError(t, err)
		defer closeDB()

		dbMockStmt := db.Session(&gorm.Session{DryRun: true}).
			Where("labelId = ?", testResourceLabel.LabelID).
			Find(&[]ResourceLabel{}).
			Statement
		dbMockArgs := modeltest.ToDriverValueSlice(dbMockStmt.Vars...)
		dbMock.ExpectQuery(regexp.QuoteMeta(dbMockStmt.SQL.String())).
			WithArgs(dbMockArgs...).
			WillReturnRows(sqlmock.NewRows(resourceLabelDBColumns).AddRows(generateResourceLabelDBRows(testResourceLabel)...))

		var results []ResourceLabel
		err = db.WithContext(context.Background()).Where("labelId = ?", testResourceLabel.LabelID).Find(&results).Error
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, testResourceLabel.AiResourceID, results[0].AiResourceID)
		assert.Equal(t, testResourceLabel.LabelID, results[0].LabelID)

		require.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("FindWithPreloadedRelations", func(t *testing.T) {
		db, dbMock, closeDB, err := modeltest.NewMockDB()
		require.NoError(t, err)
		defer closeDB()

		aiResourceDBColumns, err := modeltest.ExtractDBColumns(db, AiResource{})
		require.NoError(t, err)
		generateAiResourceDBRows, err := modeltest.NewDBRowsGenerator(db, AiResource{})
		require.NoError(t, err)

		labelDBColumns, err := modeltest.ExtractDBColumns(db, Label{})
		require.NoError(t, err)
		generateLabelDBRows, err := modeltest.NewDBRowsGenerator(db, Label{})
		require.NoError(t, err)

		testAiResource := AiResource{
			Model: Model{ID: testResourceLabel.AiResourceID},
			URL:   "https://example.com/resource.json",
		}

		testLabel := Label{
			Model:     Model{ID: testResourceLabel.LabelID},
			LabelName: "machine-learning",
		}

		dbMockStmt := db.Session(&gorm.Session{DryRun: true}).
			Preload("AiResource").
			Preload("Label").
			Where("id = ?", testResourceLabel.ID).
			First(&ResourceLabel{}).
			Statement
		dbMockArgs := modeltest.ToDriverValueSlice(dbMockStmt.Vars...)
		dbMock.ExpectQuery(regexp.QuoteMeta(dbMockStmt.SQL.String())).
			WithArgs(dbMockArgs...).
			WillReturnRows(sqlmock.NewRows(resourceLabelDBColumns).AddRows(generateResourceLabelDBRows(testResourceLabel)...))

		aiResourceStmt := db.Session(&gorm.Session{DryRun: true}).
			Where("id IN (?)", testResourceLabel.AiResourceID).
			Find(&[]AiResource{}).
			Statement
		aiResourceArgs := modeltest.ToDriverValueSlice(aiResourceStmt.Vars...)
		dbMock.ExpectQuery(regexp.QuoteMeta(aiResourceStmt.SQL.String())).
			WithArgs(aiResourceArgs...).
			WillReturnRows(sqlmock.NewRows(aiResourceDBColumns).AddRows(generateAiResourceDBRows(testAiResource)...))

		labelStmt := db.Session(&gorm.Session{DryRun: true}).
			Where("id IN (?)", testResourceLabel.LabelID).
			Find(&[]Label{}).
			Statement
		labelArgs := modeltest.ToDriverValueSlice(labelStmt.Vars...)
		dbMock.ExpectQuery(regexp.QuoteMeta(labelStmt.SQL.String())).
			WithArgs(labelArgs...).
			WillReturnRows(sqlmock.NewRows(labelDBColumns).AddRows(generateLabelDBRows(testLabel)...))

		var result ResourceLabel
		err = db.WithContext(context.Background()).
			Preload("AiResource").
			Preload("Label").
			Where("id = ?", testResourceLabel.ID).
			First(&result).Error
		require.NoError(t, err)
		assert.Equal(t, testResourceLabel.AiResourceID, result.AiResourceID)
		assert.Equal(t, testResourceLabel.LabelID, result.LabelID)
		assert.Equal(t, testAiResource.URL, result.AiResource.URL)
		assert.Equal(t, testLabel.LabelName, result.Label.LabelName)

		require.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("Delete", func(t *testing.T) {
		db, dbMock, closeDB, err := modeltest.NewMockDB()
		require.NoError(t, err)
		defer closeDB()

		dbMock.ExpectBegin()

		dbMockStmt := db.Session(&gorm.Session{DryRun: true, SkipDefaultTransaction: true}).
			Delete(&testResourceLabel).
			Statement
		dbMockArgs := modeltest.ToDriverValueSlice(dbMockStmt.Vars...)
		dbMockArgs[0] = sqlmock.AnyArg() // DeletedAt
		dbMock.ExpectExec(regexp.QuoteMeta(dbMockStmt.SQL.String())).
			WithArgs(dbMockArgs...).
			WillReturnResult(sqlmock.NewResult(0, 1))

		dbMock.ExpectCommit()

		err = db.WithContext(context.Background()).Delete(&testResourceLabel).Error
		require.NoError(t, err)

		require.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("DeleteByResourceAndLabel", func(t *testing.T) {
		db, dbMock, closeDB, err := modeltest.NewMockDB()
		require.NoError(t, err)
		defer closeDB()

		dbMock.ExpectBegin()

		dbMockStmt := db.Session(&gorm.Session{DryRun: true, SkipDefaultTransaction: true}).
			Where("aiResourceId = ? AND labelId = ?", testResourceLabel.AiResourceID, testResourceLabel.LabelID).
			Delete(&ResourceLabel{}).
			Statement
		dbMockArgs := modeltest.ToDriverValueSlice(dbMockStmt.Vars...)
		dbMockArgs[0] = sqlmock.AnyArg() // DeletedAt
		dbMock.ExpectExec(regexp.QuoteMeta(dbMockStmt.SQL.String())).
			WithArgs(dbMockArgs...).
			WillReturnResult(sqlmock.NewResult(0, 1))

		dbMock.ExpectCommit()

		err = db.WithContext(context.Background()).
			Where("aiResourceId = ? AND labelId = ?", testResourceLabel.AiResourceID, testResourceLabel.LabelID).
			Delete(&ResourceLabel{}).Error
		require.NoError(t, err)

		require.NoError(t, dbMock.ExpectationsWereMet())
	})
}

func TestResourceLabelValidation(t *testing.T) {
	t.Run("ValidResourceLabel", func(t *testing.T) {
		rl := ResourceLabel{
			AiResourceID: 123,
			LabelID:      456,
		}
		assert.Greater(t, rl.AiResourceID, int64(0))
		assert.Greater(t, rl.LabelID, int64(0))
	})

	t.Run("ZeroIDs", func(t *testing.T) {
		rl := ResourceLabel{
			AiResourceID: 0,
			LabelID:      0,
		}
		assert.Equal(t, int64(0), rl.AiResourceID)
		assert.Equal(t, int64(0), rl.LabelID)
	})

	t.Run("NegativeIDs", func(t *testing.T) {
		rl := ResourceLabel{
			AiResourceID: -1,
			LabelID:      -1,
		}
		assert.Less(t, rl.AiResourceID, int64(0))
		assert.Less(t, rl.LabelID, int64(0))
	})

	t.Run("MixedIDs", func(t *testing.T) {
		rl := ResourceLabel{
			AiResourceID: 123,
			LabelID:      0,
		}
		assert.Greater(t, rl.AiResourceID, int64(0))
		assert.Equal(t, int64(0), rl.LabelID)
	})
}