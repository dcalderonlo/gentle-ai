package shellinstaller

import (
	"fmt"
	"os"
	"strings"
)

// userHomebrewPrefixes are the shared Homebrew trees whose admin-group 0775
// directories the supervisor provenance check refuses as ancestors (#5814
// keeps that refusal). Paths use "/" on every OS so the rule stays testable.
var userHomebrewPrefixes = []string{"/opt/homebrew", "/usr/local/Cellar", "/usr/local/Homebrew"}

const userHomebrewRemediation = "; gentle-ai was started from Homebrew's shared prefix, whose group-writable directories it does not trust as a supervisor ancestor. " +
	"Private supervisor staging can acquire an authenticated release for the exact running version; earlier execution-boundary checks may still refuse this source. If acquisition is unavailable or an earlier check refuses, keep Homebrew as it is (do not change Homebrew permissions) and run gentle-ai from a private directory instead: " +
	"either `go install github.com/gentleman-programming/gentle-ai/v5/cmd/gentle-ai@v5.0.0` into your own GOBIN, " +
	"or copy the gentle-ai binary into a user-owned 0755 directory (for example ~/.local/bin) and run it from there"

// userHomebrewHint returns the remediation appended to a supervisor ancestor
// refusal when the refused path lies in a Homebrew prefix, and "" otherwise.
func userHomebrewHint(path string) string {
	for _, prefix := range userHomebrewPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return userHomebrewRemediation
		}
	}
	return ""
}

type supervisorAncestorRefusal struct {
	path    string
	message string
}

func (e *supervisorAncestorRefusal) Error() string { return e.message }

// userSupervisorAncestorRefusal is the action-time refusal of an unsafe
// supervisor ancestor; a Homebrew ancestor adds its private-directory hint.
func userSupervisorAncestorRefusal(path string, owner uint32, mode os.FileMode) error {
	return privateError("source", &supervisorAncestorRefusal{path: path, message: fmt.Sprintf("supervisor ancestor refused: path=%q owner=%d mode=%#o%s", path, owner, mode, userHomebrewHint(path))})
}
