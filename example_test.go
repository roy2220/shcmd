package shcmd_test

import (
	"fmt"
	"os"

	"github.com/roy2220/shcmd"
)

func Example() {
	os.Setenv("FOO_BAR", "foo bar")

	text := []byte(`

		ls -alh /

		grep  -F \
			'Little '\''s Law'  \
			foo.txt  # comment1

		# comment2
		ps aux

		echo ${FOO_BAR} "\"${FOO_BAR}\""

		echo $'\061\x32\x33' $'Angry\nBirds'

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
	// ["grep" "-F" "Little 's Law" "foo.txt"]
	// ["ps" "aux"]
	// ["echo" "foo" "bar" "\"foo bar\""]
	// ["echo" "123" "Angry\nBirds"]
	// processed line count: 15
}
