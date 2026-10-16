package shcmd_test

import (
	"testing"

	. "github.com/roy2220/shcmd"
	"github.com/stretchr/testify/require"
)

func TestSplit(t *testing.T) {
	command, err := Split(`echo $FOO "done"`, Options{
		ExpandEnvVars: true,
		Getenv: func(name string) string {
			return "foo bar"
		},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"echo", "foo", "bar", "done"}, command)
}

func TestSplitBytes(t *testing.T) {
	command, err := SplitBytes([]byte("echo 'hello world' # comment\nps aux"), Options{})
	require.NoError(t, err)
	require.Equal(t, []string{"echo", "hello world"}, command)
}
