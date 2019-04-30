package store

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/stretchr/testify/require"
)

func TestSystemValue(t *testing.T) {
	t.Run("unknown value", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := MakeTestSQLStore(t, logger)

		value, err := sqlStore.getSystemValue(sqlStore.db, "unknown")
		require.NoError(t, err)
		require.Empty(t, value)
	})

	t.Run("known value", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := MakeTestSQLStore(t, logger)

		key1 := "key1"
		value1 := "value1"
		key2 := "key2"
		value2 := "value2"

		err := sqlStore.setSystemValue(sqlStore.db, key1, value1)
		require.NoError(t, err)

		err = sqlStore.setSystemValue(sqlStore.db, key2, value2)
		require.NoError(t, err)

		actualValue1, err := sqlStore.getSystemValue(sqlStore.db, key1)
		require.NoError(t, err)
		require.Equal(t, value1, actualValue1)

		actualValue2, err := sqlStore.getSystemValue(sqlStore.db, key2)
		require.NoError(t, err)
		require.Equal(t, value2, actualValue2)
	})
}
