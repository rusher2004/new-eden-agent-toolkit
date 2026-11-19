package esi

import (
	"fmt"
	"net/http"
	"unicode/utf8"

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
//
// When no ESI error detail can be extracted -- most importantly when a 2xx
// response failed to decode into the generated model (e.g. "no value given
// for required property listings") -- the raw response body is appended
// instead, truncated to maxErrorBodyBytes. Without it, a decode failure
// reports only what the generated model expected, never what ESI actually
// sent, which left a transient Paragon Hub failure in the 2026-11-18 audit
// undiagnosable after the fact.
func describeError(err error, resp *http.Response) error {
	if err == nil {
		return nil
	}

	var extra string
	if oaErr, ok := err.(*esiapi.GenericOpenAPIError); ok {
		if esiErr, ok := oaErr.Model().(esiapi.Error); ok && esiErr.Error != "" {
			extra = esiErr.Error
		} else if body := oaErr.Body(); len(body) > 0 {
			extra = "response body: " + truncateBody(body)
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

// maxErrorBodyBytes caps how much of a raw response body describeError
// includes, so a large undecodable payload (a full page of listings, say)
// can't flood a tool error.
const maxErrorBodyBytes = 512

// truncateBody quotes body for inclusion in an error message, truncated to
// maxErrorBodyBytes with a marker noting how much was cut.
func truncateBody(body []byte) string {
	if len(body) <= maxErrorBodyBytes {
		return fmt.Sprintf("%q", body)
	}
	// Back off to a rune boundary so a multi-byte character straddling the
	// cut isn't rendered as stray \x escapes.
	cut := maxErrorBodyBytes
	for cut > 0 && !utf8.RuneStart(body[cut]) {
		cut--
	}
	return fmt.Sprintf("%q... (%d more bytes)", body[:cut], len(body)-cut)
}
