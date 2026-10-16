// Package shcmd parses shell-style text into commands, one argument slice
// per command. It handles quoting, ANSI-C quoting, backslash escapes, line
// continuations, comments, and optional environment variable expansion,
// following basic POSIX shell rules for lexing.
package shcmd
