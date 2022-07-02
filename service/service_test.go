package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmpty(t *testing.T) {
	b := []byte("[]")

	_, err := parseStatements(b)
	require.NotNil(t, err, "didn't return empty statement")

	b = []byte("[[]]")
	_, err = parseStatements(b)
	require.NotNil(t, err, "didn't return empty statement")
}

func TestSingleStatement(t *testing.T) {
	stmt := "SELECT * FROM test"
	b := []byte(fmt.Sprintf(`["%s"]`, stmt))

	parsed, err := parseStatements(b)
	require.NoError(t, err)
	require.Equal(t, 1, len(parsed))
	require.Equal(t, parsed[0].Sql, stmt)
	require.Nil(t, parsed[0].Params)
}

func TestInvalid(t *testing.T) {
	stmts := [][]byte{
		[]byte(`["SELECT * FROM test]`),
		[]byte(`["SELECT * FROM test"`),
	}

	for _, stmt := range stmts {
		_, err := parseStatements(stmt)
		require.Equal(t, err, ErrBadJson)
	}
}
