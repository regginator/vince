package main

import (
	"slices"
	"strings"
)

// IterProvider for raw mode

type RawIter struct{}

// TODO: We can totally do this with just a one-line calculation, revise this
func (iter *RawIter) GetPasswordCount() uint64 {
	numChars := uint64(len(*RawCharset))

	var totalPwCount uint64 = 0
	for len := int(rawRangeMin); len <= int(rawRangeMax); len++ {
		var lenPwCount uint64 = 1
		for i := 0; i < len; i++ {
			lenPwCount *= numChars
		}

		totalPwCount += lenPwCount
	}

	return totalPwCount
}

func (iter *RawIter) IterPasswords() func(func(string) bool) {
	charset := strings.Split(*RawCharset, "")
	slices.Sort(charset)
	charset = slices.Compact(charset)

	numChars := len(charset)

	return func(yield func(string) bool) {
		for len := int(rawRangeMin); len <= int(rawRangeMax); len++ {
			indices := make([]int, len)

			for {
				permutation := ""
				for _, i := range indices {
					permutation += charset[i]
				}

				if !yield(permutation) {
					return
				}

				i := len - 1
				for i >= 0 && indices[i] == numChars-1 {
					i--
				}
				if i < 0 {
					break
				}

				indices[i]++
				for j := i + 1; j < len; j++ {
					indices[j] = 0
				}
			}
		}
	}
}
