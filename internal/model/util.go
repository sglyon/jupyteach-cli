package model

import "strings"

func Slugify(x string, sep string) string {
	// slugify a string. This will convert to lowercase and replace all spaces with hyphens

	// convert to lowercase
	x = strings.ToLower(x)
	// replace colon with nothing
	x = strings.ReplaceAll(x, ":", "")
	x = strings.ReplaceAll(x, "(", "")
	x = strings.ReplaceAll(x, ")", "")
	// replace spaces, parens  with hyphens
	x = strings.ReplaceAll(x, " ", sep)

	return x
}
