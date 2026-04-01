// Package skills provides embedded skill definitions for the Spektacular Skill Library.
package skills

import "embed"

//go:embed all:*.md
var FS embed.FS
