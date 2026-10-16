package shcmd_test

import (
	"os"
	"strings"
	"testing"

	"github.com/google/shlex"
	"github.com/kballard/go-shellquote"
	"github.com/mattn/go-shellwords"
	"github.com/roy2220/shcmd"
	"github.com/stretchr/testify/require"
)

func BenchmarkSplit(b *testing.B) {
	testCases := []struct {
		Name  string
		Input string
	}{
		{"Simple", "ls -alh /var/log/messages"},
		{"Quoted", `grep -F 'Little '\''s Law' "my\$notes.txt" word\ with\ spaces`},
		{"ManyArgs", strings.Repeat("--flag=value ", 32) + "tail"},
	}

	splitters := []struct {
		Name  string
		Split func(string) ([]string, error)
	}{
		{"shcmd", func(text string) ([]string, error) { return shcmd.Split(text, shcmd.Options{}) }},
		{"go-shellquote", shellquote.Split},
		{"go-shellwords", shellwords.Parse},
		{"google-shlex", shlex.Split},
	}

	for _, tc := range testCases {
		var firstOutput []string
		for i, s := range splitters {
			output, err := s.Split(tc.Input)
			require.NoError(b, err)
			if i == 0 {
				firstOutput = output
			} else {
				require.Equal(b, firstOutput, output, "inconstant output from %s", s.Name)
			}
		}
	}

	for _, tc := range testCases {
		for _, s := range splitters {
			b.Run(tc.Name+"/"+s.Name, func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					s.Split(tc.Input)
				}
			})
		}
	}
}

func BenchmarkSplitWithEnvVarExpansion(b *testing.B) {
	os.Setenv("FOO_BAR", "foo bar")
	os.Setenv("BAZ_QUX", "baz-qux")

	testCases := []struct {
		Name  string
		Input string
	}{
		{"Simple", "echo $FOO_BAR"},
		{"Quoted", `echo "${FOO_BAR} $FOO_BAR ${BAZ_QUX}"`},
		{"ManyArgs", strings.Repeat("${FOO_BAR} ", 32) + "tail"},
	}

	shellwordsParser := shellwords.NewParser()
	shellwordsParser.ParseEnv = true
	// github.com/kballard/go-shellquote and github.com/google/shlex do not
	// support environment variable expansion.
	splitters := []struct {
		Name  string
		Split func(string) ([]string, error)
	}{
		{"shcmd", func(text string) ([]string, error) { return shcmd.Split(text, shcmd.Options{ExpandEnvVars: true}) }},
		{"go-shellwords", shellwordsParser.Parse},
	}

	for _, tc := range testCases {
		var firstOutput []string
		for i, s := range splitters {
			output, err := s.Split(tc.Input)
			require.NoError(b, err)
			if i == 0 {
				firstOutput = output
			} else {
				require.Equal(b, firstOutput, output, "inconstant output from %s", s.Name)
			}
		}
	}

	for _, tc := range testCases {
		for _, s := range splitters {
			b.Run(tc.Name+"/"+s.Name, func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					s.Split(tc.Input)
				}
			})
		}
	}
}
