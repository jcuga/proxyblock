package vars

// Variables common to all of proxyblock
// Currently here as a way around cyclic dependecies and passing around tons of
// state.  TODO: put these where they belong once refactoring is done.

import (
	"regexp"
)

var (
	StartBodyTagMatcher = regexp.MustCompile(`(?i:<body.*>)`)
	ProxyControlPort         = "8380"
	// TODO: make this UUID generated on startup, accessed via singleton?
	ProxyExceptionString = "LOL-WHUT-JUST-DOIT-DOOD"
)
