package tokensource

import "os"

var debug bool

func init() {
	debug = os.Getenv("DEBUG") != ""
}
