# shcmd

Package shcmd parses shell-style command text into commands, one
argument slice per command. It handles quoting, backslash escapes,
line continuations, comments, and optional environment variable expansion,
following basic POSIX shell rules for lexing.

## Usage

Example:

```go
package main

import (
	"fmt"
	"os"

	"github.com/roy2220/shcmd"
)

func main() {
	os.Setenv("FOOBAR", "foo bar")
	os.Setenv("EMPTY", "")

	text := []byte(`# comment1
		ls -alh /

		grep  -F \
			'hello world'  \
			foo.txt  # comment2

		# comment3
		ps aux

		echo ${FOOBAR} "${FOOBAR}"

		${EMPTY}#non-comment
`)

	scanner := shcmd.Scanner{
		ExpandEnvVar: true,
	}

	scanner.SetText(text)

	for {
		command, found, err := scanner.NextCommand()
		if err != nil {
			panic(err)
		}
		if !found {
			break
		}
		fmt.Printf("%q\n", command)
	}

	fmt.Printf("processed line count: %v\n", scanner.LineCount())
}
```

Output:

```
["ls" "-alh" "/"]
["grep" "-F" "hello world" "foo.txt"]
["ps" "aux"]
["echo" "foo" "bar" "foo bar"]
["#non-comment"]
processed line count: 13
```
