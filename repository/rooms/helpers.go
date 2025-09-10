package roomsrepository

import (
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func sortUserIDs(id1, id2 int) (int, int) {
	if id1 < id2 {
		return id1, id2
	}
	return id2, id1
}

func removeAccents(s string) (string, error) {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, err := transform.String(t, s)
	return result, err
}
