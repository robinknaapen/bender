package app

import (
	_ "embed"
)

// testServicePage is the built-in "Test" service shown in debug builds:
// buttons to exercise the notification pipe (instant and delayed, so you
// can switch away first) and the title-based badge parsing.
//
//go:embed testpage.html
var testServicePage string
