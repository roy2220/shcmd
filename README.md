# shcmd

[![Go Reference](https://pkg.go.dev/badge/github.com/roy2220/shcmd.svg)](https://pkg.go.dev/github.com/roy2220/shcmd)
[![Coverage](./.badges/coverage.svg)](#)

Package `shcmd` parses shell-style command text into commands, one
argument slice per command. It handles quoting, backslash escapes,
line continuations, comments, and optional environment variable expansion,
following basic POSIX shell rules for lexing.

> [!NOTE]
> Except for the documentation, every line of code is human-crafted. **No AI slop included**.

## Usage

### Example: single command

```go
package main

import (
    "fmt"
    "os"

    "github.com/roy2220/shcmd"
)

func main() {
    os.Setenv("MY_FILE", "my notes.txt")

    command, err := shcmd.Split(
        `grep  -F  \
            'Little '\''s Law'  \
            "${MY_FILE}"  # comment
        `,
        shcmd.Options{ExpandEnvVars: true},
    )
    if err != nil {
        panic(err)
    }
    fmt.Printf("%q\n", command)

    // Output:
    // ["grep" "-F" "Little 's Law" "my notes.txt"]
}
```

To parse from a `[]byte` without copying the text into a string,
use `SplitBytes`; it takes the same options and returns the same results.

### Example: multiple commands

```go
package main

import (
    "fmt"
    "os"

    "github.com/roy2220/shcmd"
)

func main() {
    os.Setenv("FOO_BAR", "foo bar")

    text := []byte(`

        ls -alh /

        sed -i \
            's/apple/orange/g' \
            test.txt

        # comment
        ps aux

        echo ${FOO_BAR} "\"${FOO_BAR}\""

        echo $'\061\x32\x33' $'Angry\nBirds'

`)

    scanner := shcmd.Scanner{
        Options: shcmd.Options{
            ExpandEnvVars: true,
        },
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

    // Output:
    // ["ls" "-alh" "/"]
    // ["sed" "-i" "s/apple/orange/g" "test.txt"]
    // ["ps" "aux"]
    // ["echo" "foo" "bar" "\"foo bar\""]
    // ["echo" "123" "Angry\nBirds"]
    // processed line count: 15
}
```

## Limitations

It is a shell-word lexer, not a shell parser: it supports only a basic subset of POSIX rules. The following rules are not supported:

```sh
ls|grep foo; ps aux && true   # one command: ["ls|grep" "foo;" "ps" "aux" "&&" "true"]
cmd >out 2>&1 <in             # redirection is not special: ["cmd" ">out" "2>&1" "<in"]
cat <<EOF                     # no here-documents: ["cat" "<<EOF"]
echo $(date) `date`           # no command substitution: ["echo" "$(date)" "`date`"]
echo $1 $? $$ $@              # no positional or special parameters: ["echo" "$1" "$?" "$$" "$@"]
echo $'\cA\u00e9'             # \cX and \u are not decoded: ["echo" "\\cA\\u00e9"]
echo ${VAR:-default}          # bad substitution: only $NAME and ${NAME} are expanded
ls ~/*.txt                    # no tilde expansion or globbing: ["ls" "~/*.txt"]
```

## License

MIT
