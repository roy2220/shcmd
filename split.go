package shcmd

import "sync"

var scannerPool = sync.Pool{New: func() any { return new(Scanner) }}

// Split parses and returns the first command in the text as a slice of
// arguments, with each argument fully unquoted and unescaped. Blank lines and
// comments are skipped. Any subsequent commands are discarded.
//
// If no command was found, an empty slice is returned.
func Split(text string, options Options) ([]string, error) {
	return SplitBytes(s2b(text), options)
}

// SplitBytes parses and returns the first command in the text as a slice of
// arguments, with each argument fully unquoted and unescaped. Blank lines and
// comments are skipped. Any subsequent commands are discarded.
//
// If no command was found, an empty slice is returned.
func SplitBytes(text []byte, options Options) ([]string, error) {
	scanner := scannerPool.Get().(*Scanner)
	defer scannerPool.Put(scanner)

	scanner.Options = options
	scanner.SetText(text)
	command, _, err := scanner.NextCommand()
	return command, err
}
