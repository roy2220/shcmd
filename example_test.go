package shcmd_test

import (
	"fmt"
	"os"

	"github.com/roy2220/shcmd"
)

func Example() {
	os.Setenv("FOOBAR", "foo bar")

	text := []byte(`

		ls -alh /

		grep  -F \
			'hello world'  \
			foo.txt  # comment1

		# comment2
		ps aux

		echo ${FOOBAR} "${FOOBAR}"

		echo $'\x41ngry\nBirds'

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

	// Output:
	// ["ls" "-alh" "/"]
	// ["grep" "-F" "hello world" "foo.txt"]
	// ["ps" "aux"]
	// ["echo" "foo" "bar" "foo bar"]
	// ["echo" "Angry\nBirds"]
	// processed line count: 15
}
