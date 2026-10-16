package shcmd_test

import (
	"fmt"
	"os"

	"github.com/roy2220/shcmd"
)

func Example_single_command() {
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

func Example_multiple_commands() {
	os.Setenv("FOO_BAR", "foo bar")

	text := []byte(`

		ls -alh /

		sed -i \
			's/apple/orange/g' \
			test.txt

		# comment
		ps aux

		echo ${FOO_BAR} "${FOO_BAR}"

		echo $'have fun\t\U0001F920'

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
	// ["echo" "foo" "bar" "foo bar"]
	// ["echo" "have fun\t🤠"]
	// processed line count: 16
}
