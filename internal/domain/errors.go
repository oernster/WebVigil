// Package domain holds WebVigil's rules: what a site is, how content is compared, how a check
// outcome becomes a status and how the exchange format is read. It performs no I/O and reads no
// clock; every time it needs arrives as an argument.
package domain

import "errors"

// Sentinel errors. Callers wrap them with detail and test them with errors.Is.
var (
	ErrInvalidURL      = errors.New("invalid URL")
	ErrInvalidSite     = errors.New("invalid site")
	ErrInvalidPattern  = errors.New("invalid exclusion pattern")
	ErrImportTooLarge  = errors.New("import file too large")
	ErrImportMalformed = errors.New("import file is not valid JSON")
	ErrImportFormat    = errors.New("not a WebVigil sites file")
	ErrImportSchema    = errors.New("unsupported schema version")
	ErrImportTooMany   = errors.New("too many sites in import file")
	ErrInvalidSettings = errors.New("invalid settings")
	ErrSelectorMissing = errors.New("a required selector matched nothing")
	ErrParseFailure    = errors.New("the page could not be parsed")
)
