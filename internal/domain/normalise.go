package domain

import (
	"fmt"
	"regexp"
	"strings"
)

var horizontalSpace = regexp.MustCompile(`[ \t\f\v\x{00A0}]+`)

// Normalise applies the ignore rules to extracted text so that only meaningful differences
// survive: line endings become LF, exclusion patterns are removed, trailing space on each line
// is trimmed, whitespace is collapsed when asked, case is folded when asked and leading or
// trailing blank lines are dropped.
func Normalise(text string, rules IgnoreRules) (string, error) {
	patterns, err := compilePatterns(rules.ExcludePatterns)
	if err != nil {
		return "", err
	}
	return normalise(text, patterns, rules), nil
}

// NormaliseContent applies Normalise to the text and to every field value.
func NormaliseContent(c Content, rules IgnoreRules) (Content, error) {
	patterns, err := compilePatterns(rules.ExcludePatterns)
	if err != nil {
		return Content{}, err
	}
	fields := make([]FieldValue, len(c.Fields))
	for i, f := range c.Fields {
		fields[i] = FieldValue{Name: f.Name, Value: normalise(f.Value, patterns, rules)}
	}
	return Content{Text: normalise(c.Text, patterns, rules), Fields: fields}, nil
}

func compilePatterns(src []string) ([]*regexp.Regexp, error) {
	out := make([]*regexp.Regexp, 0, len(src))
	for _, p := range src {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("%w: %q: %v", ErrInvalidPattern, p, err)
		}
		out = append(out, re)
	}
	return out, nil
}

func normalise(text string, patterns []*regexp.Regexp, rules IgnoreRules) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	for _, re := range patterns {
		text = re.ReplaceAllString(text, "")
	}
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if rules.CollapseWhitespace {
			line = strings.TrimSpace(horizontalSpace.ReplaceAllString(line, " "))
			if line == "" {
				continue
			}
		} else {
			line = strings.TrimRight(line, " \t\f\v")
		}
		out = append(out, line)
	}
	result := strings.Trim(strings.Join(out, "\n"), "\n")
	if rules.CaseInsensitive {
		result = strings.ToLower(result)
	}
	return result
}
