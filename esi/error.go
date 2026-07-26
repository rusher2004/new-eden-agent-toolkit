package esi

import (
	"fmt"
	"net/http"

	esiapi "github.com/fnt-eve/goesi-openapi/esi"
)

// describeError enriches err with ESI's actual error detail and, when
// present, its error-limit rate headers. This exists because
// goesi-openapi's generated error type degrades to a bare HTTP status
// string on its own: its formatErrorMessage helper looks for Title/Detail
// fields (an RFC 7807-style convention) via reflection, but ESI's actual
// error model (esiapi.Error) has Error/Details fields instead, so that
// lookup always misses and the real message is silently lost unless
// callers dig it out of the error's Model() themselves.
//
// Every esi/ method that calls .Execute() on a generated API request MUST
// route its error and *http.Response through this function before
// wrapping and returning it — see CLAUDE.md's "ESI error detail" hard
// rule. Passing a nil resp (e.g. when the request never reached the
// network) is fine; only the error-detail extraction runs in that case.
func describeError(err error, resp *http.Response) error {
	if err == nil {
		return nil
	}

	var extra string
	if oaErr, ok := err.(*esiapi.GenericOpenAPIError); ok {
		if esiErr, ok := oaErr.Model().(esiapi.Error); ok && esiErr.Error != "" {
			extra = esiErr.Error
		}
	}

	if resp != nil {
		if remain := resp.Header.Get("X-Esi-Error-Limit-Remain"); remain != "" {
			reset := resp.Header.Get("X-Esi-Error-Limit-Reset")
			if extra != "" {
				extra += "; "
			}
			extra += fmt.Sprintf("error limit remaining: %s, resets in %ss", remain, reset)
		}
	}

	if extra == "" {
		return err
	}
	return fmt.Errorf("%s (%w)", extra, err)
}
