package shcmd_test

import (
	"fmt"
	"runtime"
	"testing"

	. "github.com/roy2220/shcmd"
	"github.com/stretchr/testify/require"
)

func lineno() int {
	_, _, l, _ := runtime.Caller(1)
	return l
}

func TestPositive(t *testing.T) {
	t.Setenv("FOO", "foo")
	t.Setenv("FOOBAR", "foo bar")
	t.Setenv("SPACED", "  leading and trailing  ")
	t.Setenv("EMPTY", "")

	testCases := []struct {
		Lineno int

		Text string

		Command []string
	}{
		{
			Lineno:  lineno(),
			Text:    "foo bar",
			Command: []string{"foo", "bar"},
		},
		{
			Lineno:  lineno(),
			Text:    "foo   \t bar\tbaz",
			Command: []string{"foo", "bar", "baz"},
		},
		{
			Lineno:  lineno(),
			Text:    "   ls\t-alh  /   ",
			Command: []string{"ls", "-alh", "/"},
		},
		{
			Lineno:  lineno(),
			Text:    "ls -alh /",
			Command: []string{"ls", "-alh", "/"},
		},
		{
			Lineno:  lineno(),
			Text:    "echo 'hello world' foo",
			Command: []string{"echo", "hello world", "foo"},
		},
		{
			Lineno:  lineno(),
			Text:    "echo \"hello world\" foo",
			Command: []string{"echo", "hello world", "foo"},
		},
		{
			Lineno:  lineno(),
			Text:    "echo both'to''gether'",
			Command: []string{"echo", "bothtogether"},
		},
		{
			Lineno:  lineno(),
			Text:    "cmd a\"b\"c",
			Command: []string{"cmd", "abc"},
		},
		{
			Lineno:  lineno(),
			Text:    "cmd a'b c'd",
			Command: []string{"cmd", "ab cd"},
		},
		{
			Lineno:  lineno(),
			Text:    "cmd '' \"\" x",
			Command: []string{"cmd", "", "", "x"},
		},
		{
			Lineno:  lineno(),
			Text:    "cmd a\\ b",
			Command: []string{"cmd", "a b"},
		},
		{
			Lineno:  lineno(),
			Text:    "cmd a\\b",
			Command: []string{"cmd", "ab"},
		},
		{
			Lineno:  lineno(),
			Text:    "echo \\$FOO",
			Command: []string{"echo", "$FOO"},
		},
		{
			Lineno:  lineno(),
			Text:    "cmd \"escaped \\\" quote\"",
			Command: []string{"cmd", "escaped \" quote"},
		},
		{
			Lineno:  lineno(),
			Text:    "cmd \"keep \\\\ literal\"",
			Command: []string{"cmd", "keep \\ literal"},
		},
		{
			Lineno:  lineno(),
			Text:    "cmd \"keep \\n literal\"",
			Command: []string{"cmd", "keep \\n literal"},
		},
		{
			Lineno:  lineno(),
			Text:    "cmd 'raw \\ backslash'",
			Command: []string{"cmd", "raw \\ backslash"},
		},
		{
			Lineno:  lineno(),
			Text:    "echo `date`",
			Command: []string{"echo", "`date`"},
		},
		{
			Lineno: lineno(),
			Text: `grep  -F \
'hello world' \
foo.txt`,
			Command: []string{"grep", "-F", "hello world", "foo.txt"},
		},
		{
			Lineno: lineno(),
			Text: `cmd a\
b`,
			Command: []string{"cmd", "ab"},
		},
		{
			Lineno: lineno(),
			Text: `cmd "line \
continued" done`,
			Command: []string{"cmd", "line continued", "done"},
		},
		{
			Lineno: lineno(),
			Text: `cmd "first
second" done`,
			Command: []string{"cmd", "first\nsecond", "done"},
		},
		{
			Lineno: lineno(),
			Text: `# comment
ls -alh /`,
			Command: []string{"ls", "-alh", "/"},
		},
		{
			Lineno:  lineno(),
			Text:    "ls -alh / # trailing comment",
			Command: []string{"ls", "-alh", "/"},
		},
		{
			Lineno:  lineno(),
			Text:    "ls -alh /#not-a-comment",
			Command: []string{"ls", "-alh", "/#not-a-comment"},
		},
		{
			Lineno:  lineno(),
			Text:    "ls '#not-a-comment'",
			Command: []string{"ls", "#not-a-comment"},
		},
		{
			Lineno:  lineno(),
			Text:    "ls -alh /\r\n",
			Command: []string{"ls", "-alh", "/"},
		},
		{
			Lineno:  lineno(),
			Text:    "echo $FOO",
			Command: []string{"echo", "foo"},
		},
		{
			Lineno:  lineno(),
			Text:    "echo ${FOO}",
			Command: []string{"echo", "foo"},
		},
		{
			Lineno:  lineno(),
			Text:    "echo \"$FOO\"",
			Command: []string{"echo", "foo"},
		},
		{
			Lineno:  lineno(),
			Text:    "echo '$FOO'",
			Command: []string{"echo", "$FOO"},
		},
		{
			Lineno:  lineno(),
			Text:    "echo \"\\$FOO\"",
			Command: []string{"echo", "$FOO"},
		},
		{
			Lineno:  lineno(),
			Text:    "echo $FOOBAR",
			Command: []string{"echo", "foo", "bar"},
		},
		{
			Lineno:  lineno(),
			Text:    "echo ${FOOBAR}",
			Command: []string{"echo", "foo", "bar"},
		},
		{
			Lineno:  lineno(),
			Text:    "echo \"$FOOBAR\"",
			Command: []string{"echo", "foo bar"},
		},
		{
			Lineno:  lineno(),
			Text:    "echo x${FOO}y",
			Command: []string{"echo", "xfooy"},
		},
		{
			Lineno:  lineno(),
			Text:    "echo ${FOOBAR}x",
			Command: []string{"echo", "foo", "barx"},
		},
		{
			Lineno:  lineno(),
			Text:    "echo ${EMPTY}",
			Command: []string{"echo"},
		},
		{
			Lineno:  lineno(),
			Text:    "x${EMPTY}y",
			Command: []string{"xy"},
		},
		{
			Lineno:  lineno(),
			Text:    "${EMPTY}#non-comment",
			Command: []string{"#non-comment"},
		},
		{
			Lineno:  lineno(),
			Text:    "echo $SPACED",
			Command: []string{"echo", "leading", "and", "trailing"},
		},
		{
			Lineno:  lineno(),
			Text:    "echo prefix${SPACED}suffix",
			Command: []string{"echo", "prefix", "leading", "and", "trailing", "suffix"},
		},
	}

	scanner := Scanner{
		ExpandEnvVar: true,
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("line_%d", tc.Lineno), func(t *testing.T) {
			scanner.SetText([]byte(tc.Text))
			command, found, err := scanner.NextCommand()
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, tc.Command, command)
		})
	}
}

func TestNegative(t *testing.T) {
	testCases := []struct {
		Lineno int

		Text string

		Error string
	}{
		{
			Lineno: lineno(),
			Text:   "",
		},
		{
			Lineno: lineno(),
			Text:   "   ",
		},
		{
			Lineno: lineno(),
			Text:   "\n\n",
		},
		{
			Lineno: lineno(),
			Text:   "  \t# a comment\n",
		},
		{
			Lineno: lineno(),
			Text:   "#",
		},
		{
			Lineno: lineno(),
			Text: `\
`,
		},
		{
			Lineno: lineno(),
			Text:   " '  ",
			Error:  "unterminated single-quoted string",
		},
		{
			Lineno: lineno(),
			Text:   "'",
			Error:  "unterminated single-quoted string",
		},
		{
			Lineno: lineno(),
			Text:   "echo 'hello",
			Error:  "unterminated single-quoted string",
		},
		{
			Lineno: lineno(),
			Text: `echo 'spans
lines`,
			Error: "unterminated single-quoted string",
		},
		{
			Lineno: lineno(),
			Text:   "\"",
			Error:  "unterminated double-quoted string",
		},
		{
			Lineno: lineno(),
			Text:   "echo \"hello",
			Error:  "unterminated double-quoted string",
		},
		{
			Lineno: lineno(),
			Text: `echo "spans
lines`,
			Error: "unterminated double-quoted string",
		},
		{
			Lineno: lineno(),
			Text:   `echo "escaped \"`,
			Error:  "unterminated double-quoted string",
		},
		{
			Lineno: lineno(),
			Text:   "\\",
			Error:  "unterminated backslash-escape",
		},
		{
			Lineno: lineno(),
			Text:   "cmd a\\",
			Error:  "unterminated backslash-escape",
		},
		{
			Lineno: lineno(),
			Text:   "cmd \"a\\",
			Error:  "unterminated backslash-escape",
		},
		{
			Lineno: lineno(),
			Text:   "echo ${}",
			Error:  "bad substitution",
		},
		{
			Lineno: lineno(),
			Text:   "echo ${1FOO}",
			Error:  "bad substitution",
		},
		{
			Lineno: lineno(),
			Text:   "echo ${FOO",
			Error:  "bad substitution",
		},
		{
			Lineno: lineno(),
			Text:   "echo ${FOO bar}",
			Error:  "bad substitution",
		},
		{
			Lineno: lineno(),
			Text:   "echo ${FOO!}",
			Error:  "bad substitution",
		},
		{
			Lineno: lineno(),
			Text:   "echo \"${}\"",
			Error:  "bad substitution",
		},
		{
			Lineno: lineno(),
			Text:   "echo ${",
			Error:  "bad substitution",
		},
		{
			Lineno: lineno(),
			Text:   "echo \"${a",
			Error:  "bad substitution",
		},
		{
			Lineno: lineno(),
			Text:   "echo \"${ab",
			Error:  "bad substitution",
		},
		{
			Lineno: lineno(),
			Text:   "echo \"${_",
			Error:  "bad substitution",
		},
		{
			Lineno: lineno(),
			Text:   "echo \"${a1",
			Error:  "bad substitution",
		},
		{
			Lineno: lineno(),
			Text:   "echo \"${a_1",
			Error:  "bad substitution",
		},
		{
			Lineno: lineno(),
			Text:   "echo \"$",
			Error:  "unterminated double-quoted string",
		},
		{
			Lineno: lineno(),
			Text:   "echo \"$1",
			Error:  "unterminated double-quoted string",
		},
	}

	scanner := Scanner{
		ExpandEnvVar: true,
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("line_%d", tc.Lineno), func(t *testing.T) {
			scanner.SetText([]byte(tc.Text))
			command, found, err := scanner.NextCommand()
			require.Empty(t, command)
			require.False(t, found)
			if tc.Error == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.Error)
			}
		})
	}
}

func TestSkipLines(t *testing.T) {
	t.Run("skip_none", func(t *testing.T) {
		scanner := Scanner{}
		scanner.SetText([]byte(`ls -alh /
ps aux
`))
		scanner.SkipLines(0)
		command, found, err := scanner.NextCommand()
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, []string{"ls", "-alh", "/"}, command)
		require.Equal(t, 1, scanner.LineCount())
	})

	t.Run("skip_to_middle", func(t *testing.T) {
		scanner := Scanner{}
		scanner.SetText([]byte(`ls -alh /
ps aux
whoami
`))
		scanner.SkipLines(1)
		command, found, err := scanner.NextCommand()
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, []string{"ps", "aux"}, command)
		require.Equal(t, 2, scanner.LineCount())
	})

	t.Run("skip_to_end", func(t *testing.T) {
		scanner := Scanner{}
		scanner.SetText([]byte(`ls -alh /
ps aux
`))
		scanner.SkipLines(2)
		command, found, err := scanner.NextCommand()
		require.NoError(t, err)
		require.False(t, found)
		require.Empty(t, command)
		require.Equal(t, 2, scanner.LineCount())
	})

	t.Run("skip_past_end", func(t *testing.T) {
		scanner := Scanner{}
		scanner.SetText([]byte(`ls -alh /
ps aux
`))
		scanner.SkipLines(5)
		command, found, err := scanner.NextCommand()
		require.NoError(t, err)
		require.False(t, found)
		require.Empty(t, command)
		require.Equal(t, 2, scanner.LineCount())
	})

	t.Run("last_line_without_newline", func(t *testing.T) {
		scanner := Scanner{}
		scanner.SetText([]byte(`ls -alh /
ps aux
whoami`))
		scanner.SkipLines(2)
		command, found, err := scanner.NextCommand()
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, []string{"whoami"}, command)
		require.Equal(t, 3, scanner.LineCount())
	})
}

func TestExpandEnvVar(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		t.Setenv("SHCMD_TEST_FOO", "foo bar")
		var names []string
		scanner := Scanner{
			Getenv: func(name string) string {
				names = append(names, name)
				return "from getenv"
			},
		}
		scanner.SetText([]byte(`echo $SHCMD_TEST_FOO ${SHCMD_TEST_FOO} "$SHCMD_TEST_FOO"`))
		command, found, err := scanner.NextCommand()
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, []string{"echo", "$SHCMD_TEST_FOO", "${SHCMD_TEST_FOO}", "$SHCMD_TEST_FOO"}, command)
		require.Empty(t, names)
	})

	t.Run("enabled_process_env", func(t *testing.T) {
		t.Setenv("SHCMD_TEST_FOO", "foo bar")
		scanner := Scanner{
			ExpandEnvVar: true,
		}
		scanner.SetText([]byte(`echo $SHCMD_TEST_FOO "${SHCMD_TEST_FOO}"`))
		command, found, err := scanner.NextCommand()
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, []string{"echo", "foo", "bar", "foo bar"}, command)
	})

	t.Run("enabled_injected_getenv", func(t *testing.T) {
		t.Setenv("FOO", "from process env")
		var names []string
		scanner := Scanner{
			ExpandEnvVar: true,
			Getenv: func(name string) string {
				names = append(names, name)
				return map[string]string{"FOO": "foo", "FOOBAR": "foo bar"}[name]
			},
		}
		scanner.SetText([]byte(`echo $FOO ${FOOBAR} "$FOO"`))
		command, found, err := scanner.NextCommand()
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, []string{"echo", "foo", "foo", "bar", "foo"}, command)
		require.Equal(t, []string{"FOO", "FOOBAR", "FOO"}, names)
	})
}
