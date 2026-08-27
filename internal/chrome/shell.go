package chrome

import (
	_ "embed"
	"strings"
)

// The shell document is loaded once via NavigateToString: compiled loom
// styles, the glue script, and an empty mount that Render output is
// swapped into. styles.css is produced by `make ui`.
var (
	//go:embed styles.css
	styles string
	//go:embed glue.js
	glue string
)

// Shell returns the complete chrome document.
func Shell() string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><title>bender</title><style>`)
	b.WriteString(styles)
	b.WriteString(`</style></head><body class="bg-base-100/10 text-base-800 dark:bg-base-900 dark:text-base-100 select-none"><div id="mount" class="h-screen overflow-y-auto p-2"></div><script>`)
	b.WriteString(glue)
	b.WriteString(`</script></body></html>`)
	return b.String()
}
