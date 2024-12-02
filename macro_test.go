package go_time_macro

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExpandTimeMacro(t *testing.T) {
	testTime, err := time.Parse("2006-01-02", "2023-02-28")
	require.NoError(t, err)

	{
		sql := ExpandTimeMacro("select * from table where date = ${date}", testTime)
		require.EqualValues(t, "select * from table where date = 20230228", sql)
	}
	{
		sql := ExpandTimeMacro("select * from table where date = ${date-3}", testTime)
		require.EqualValues(t, "select * from table where date = 20230225", sql)
	}
	{
		sql := ExpandTimeMacro("select * from table where date = ${date-3+1m+2d}", testTime)
		require.EqualValues(t, "select * from table where date = 20230327", sql)
	}
	{
		sql := ExpandTimeMacro("select * from table where date = ${DATE}", testTime)
		require.EqualValues(t, "select * from table where date = 2023-02-28", sql)
	}
}
