package preflight

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/joho/godotenv"
)

// EnvShadowing reports variables exported in the shell that also appear in .env with a
// different value.
//
// The loader does not override what is already exported, so the exported value silently
// wins. That is invisible at every other point in the workflow: a stale exported tag or
// cluster name changes what gets deployed with no message anywhere.
func EnvShadowing() Check {
	return Check{
		ID:    "env-shadowing",
		Title: "Settings file",
		Run: func(context.Context) Result {
			return runEnvShadowing(".env")
		},
	}
}

func runEnvShadowing(path string) Result {
	fileVars, err := godotenv.Read(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{
				Status:  StatusInfo,
				Summary: fmt.Sprintf("no %s here yet", path),
			}
		}
		return Result{
			Status:  StatusSkip,
			Summary: fmt.Sprintf("could not read %s", path),
			Detail:  err.Error(),
		}
	}

	shadowed := make([]string, 0)
	observed := map[string]string{}

	for key, fileValue := range fileVars {
		exported, present := os.LookupEnv(key)
		if !present || exported == fileValue {
			continue
		}
		shadowed = append(shadowed, key)
		observed[key] = fmt.Sprintf("exported=%s file=%s", redact(key, exported), redact(key, fileValue))
	}
	sort.Strings(shadowed)

	if len(shadowed) == 0 {
		return Result{
			Status:  StatusPass,
			Summary: fmt.Sprintf("nothing in this terminal is overriding %s", path),
		}
	}

	noun := "settings"
	verb := "override"
	if len(shadowed) == 1 {
		noun, verb = "setting", "overrides"
	}

	return Result{
		Status: StatusWarn,
		Summary: fmt.Sprintf("%d %s in this terminal %s %s: %s",
			len(shadowed), noun, verb, path, strings.Join(shadowed, ", ")),
		Detail: "Values already set in your terminal win over the file, with no message. " +
			"This is how an old version gets deployed long after the file was corrected.",
		Remedy: []string{
			"unset " + strings.Join(shadowed, " "),
			"Or open a new terminal, if these come from your shell profile.",
		},
		Observed: observed,
	}
}

// redact hides values whose name suggests a credential. The report is meant to be pasted
// into bug reports, so it must never be the reason a secret leaves the machine.
func redact(key, value string) string {
	upper := strings.ToUpper(key)
	for _, marker := range []string{"PASSWORD", "SECRET", "TOKEN", "KEY", "CREDENTIAL"} {
		if strings.Contains(upper, marker) {
			return "<redacted>"
		}
	}
	return value
}
