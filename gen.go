package main

// This directive is placed at the project root so that templ generate runs from
// the root directory. This ensures the file paths embedded in generated code are
// consistent regardless of whether generation is triggered via "go generate" or
// by running "templ generate" manually from the project root.
//go:generate go tool github.com/a-h/templ/cmd/templ generate
