# Script tests

This directory contains "script tests". When testing the bot package, all `.txt`
files in this directory will be loaded and executed as individual subtests. This
allows for fast creation of tests without needing to write them all out in Go
directly, which takes a good amount of time.

This idea is shamelessly stolen from the Go project, which uses directories of
`.go` files and `txtar` tests instead of loads of individual unit tests.
