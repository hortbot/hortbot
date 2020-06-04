// Package pkger wraps pkger to ensure a common package contains the embedded data.
// It abuses the fact that pkger's static analysis only cares that the package name matches.
package pkger

import "github.com/markbates/pkger"

type Dir = pkger.Dir
