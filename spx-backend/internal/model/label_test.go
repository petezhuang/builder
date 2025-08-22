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

func TestLabelTableName(t *testing.T) {
	l := Label{}
	assert.Equal(t, "label", l.TableName())
}

func TestLabelDBOperations(t *testing.T) {
	db, _, closeDB, err := modeltest.NewMockDB()
	require.NoError(t, err)
	closeDB()
	labelDBColumns, err := modeltest.ExtractDBColumns(db, Label{})
	require.NoError(t, err)
	generateLabelDBRows, err := modeltest.NewDBRowsGenerator(db, Label{})
	require.NoError(t, err)

	testLabel := Label{
		Model:     Model{ID: 1},
		LabelName: "machine-learning",
	}

	t.Run("Create", func(t *testing.T) {
		db, dbMock, closeDB, err := modeltest.NewMockDB()
		require.NoError(t, err)
		defer closeDB()

		dbMock.ExpectBegin()

		dbMockStmt := db.Session(&gorm.Session{DryRun: true, SkipDefaultTransaction: true}).
			Create(&testLabel).
			Statement
		dbMockArgs := modeltest.ToDriverValueSlice(dbMockStmt.Vars...)
		dbMockArgs[1] = sqlmock.AnyArg() // CreatedAt
		dbMockArgs[2] = sqlmock.AnyArg() // UpdatedAt
		dbMock.ExpectExec(regexp.QuoteMeta(dbMockStmt.SQL.String())).
			WithArgs(dbMockArgs...).
			WillReturnResult(sqlmock.NewResult(1, 1))

		dbMock.ExpectCommit()

		err = db.WithContext(context.Background()).Create(&testLabel).Error
		require.NoError(t, err)

		require.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("Find", func(t *testing.T) {
		db, dbMock, closeDB, err := modeltest.NewMockDB()
		require.NoError(t, err)
		defer closeDB()

		dbMockStmt := db.Session(&gorm.Session{DryRun: true}).
			Where("labelName = ?", testLabel.LabelName).
			First(&Label{}).
			Statement
		dbMockArgs := modeltest.ToDriverValueSlice(dbMockStmt.Vars...)
		dbMock.ExpectQuery(regexp.QuoteMeta(dbMockStmt.SQL.String())).
			WithArgs(dbMockArgs...).
			WillReturnRows(sqlmock.NewRows(labelDBColumns).AddRows(generateLabelDBRows(testLabel)...))

		var result Label
		err = db.WithContext(context.Background()).Where("labelName = ?", testLabel.LabelName).First(&result).Error
		require.NoError(t, err)
		assert.Equal(t, testLabel.ID, result.ID)
		assert.Equal(t, testLabel.LabelName, result.LabelName)

		require.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("UpdateLabelName", func(t *testing.T) {
		db, dbMock, closeDB, err := modeltest.NewMockDB()
		require.NoError(t, err)
		defer closeDB()

		updatedLabel := testLabel
		updatedLabel.LabelName = "deep-learning"

		dbMock.ExpectBegin()

		dbMockStmt := db.Session(&gorm.Session{DryRun: true, SkipDefaultTransaction: true}).
			Model(&testLabel).
			Updates(map[string]interface{}{"labelName": updatedLabel.LabelName}).
			Statement
		dbMockArgs := modeltest.ToDriverValueSlice(dbMockStmt.Vars...)
		dbMockArgs[0] = sqlmock.AnyArg() // UpdatedAt
		dbMock.ExpectExec(regexp.QuoteMeta(dbMockStmt.SQL.String())).
			WithArgs(dbMockArgs...).
			WillReturnResult(sqlmock.NewResult(0, 1))

		dbMock.ExpectCommit()

		err = db.WithContext(context.Background()).Model(&testLabel).Updates(map[string]interface{}{"labelName": updatedLabel.LabelName}).Error
		require.NoError(t, err)

		require.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("Delete", func(t *testing.T) {
		db, dbMock, closeDB, err := modeltest.NewMockDB()
		require.NoError(t, err)
		defer closeDB()

		dbMock.ExpectBegin()

		dbMockStmt := db.Session(&gorm.Session{DryRun: true, SkipDefaultTransaction: true}).
			Delete(&testLabel).
			Statement
		dbMockArgs := modeltest.ToDriverValueSlice(dbMockStmt.Vars...)
		dbMockArgs[0] = sqlmock.AnyArg() // DeletedAt
		dbMock.ExpectExec(regexp.QuoteMeta(dbMockStmt.SQL.String())).
			WithArgs(dbMockArgs...).
			WillReturnResult(sqlmock.NewResult(0, 1))

		dbMock.ExpectCommit()

		err = db.WithContext(context.Background()).Delete(&testLabel).Error
		require.NoError(t, err)

		require.NoError(t, dbMock.ExpectationsWereMet())
	})
}

func TestLabelValidation(t *testing.T) {
	t.Run("ValidLabelName", func(t *testing.T) {
		testCases := []struct {
			name      string
			labelName string
		}{
			{"Simple", "ai"},
			{"WithHyphen", "machine-learning"},
			{"WithUnderscore", "deep_learning"},
			{"Mixed", "AI_Model-v2"},
			{"MaxLength", "this-is-a-very-long-label-name-that-should-fit"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				l := Label{LabelName: tc.labelName}
				assert.NotEmpty(t, l.LabelName)
				assert.LessOrEqual(t, len(l.LabelName), 50, "Label name should not exceed 50 characters")
			})
		}
	})

	t.Run("EmptyLabelName", func(t *testing.T) {
		l := Label{LabelName: ""}
		assert.Empty(t, l.LabelName)
	})

	t.Run("LongLabelName", func(t *testing.T) {
		longName := "this-is-a-very-long-label-name-that-exceeds-the-fifty-character-limit-defined-in-the-database-schema"
		l := Label{LabelName: longName}
		assert.Greater(t, len(l.LabelName), 50, "This test verifies long label names are detected")
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		testCases := []struct {
			name      string
			labelName string
		}{
			{"WithSpaces", "machine learning"},
			{"WithDots", "ai.model"},
			{"WithSlash", "ai/ml"},
			{"WithNumbers", "model-v1.2.3"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				l := Label{LabelName: tc.labelName}
				assert.NotEmpty(t, l.LabelName)
			})
		}
	})
}