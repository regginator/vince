package main

// IterProvider impl and utils for wordlist mode

import (
	"bufio"
	"os"
	"slices"
	"strings"

	"github.com/pterm/pterm"
)

type WordlistIter struct {
	// So we aren't actually walking through the wordlist 10 gorillion times
	cachedPwCount uint64
	wordStore     []string
}

func (iter *WordlistIter) GetPasswordCount() uint64 {
	if iter.cachedPwCount != 0 {
		return iter.cachedPwCount
	}

	var count uint64 = 0
	for range iter.IterPasswords() {
		count += 1
	}

	iter.cachedPwCount = count
	return count
}

func (iter *WordlistIter) IterPasswords() func(func(string) bool) {
	f, err := os.Open(*WordlistPath)
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	// We're actually going to be evil and store the wordlist in fricking memory
	//log.Println("Reading wordlist into memory store")
	if len(iter.wordStore) == 0 {
		origWordCount := countWordsInFile(*WordlistPath)
		wordStore := make([]string, origWordCount)

		i := 0
		for scanner.Scan() {
			word := scanner.Text()
			if !lineIsWord(word) {
				continue
			}

			n := len(word)
			if n > 8 {
				n = 8
			}

			wordStore[i] = word[:n]
			i++
		}

		if err := scanner.Err(); err != nil {
			pterm.Error.Printf("error reading wordlist: %s\n", err)
		}

		// Remove duplicate entries, realloc entire word store
		wordStore = slices.Compact(wordStore)

		iter.wordStore = wordStore
	}

	return func(yield func(string) bool) {
		for i := 0; i < len(iter.wordStore); i++ {
			if !yield(iter.wordStore[i]) {
				return
			}
		}
	}
}

func lineIsWord(line string) bool {
	return strings.TrimSpace(line) != ""
}

func countWordsInFile(filePath string) int {
	out := 0

	f, err := os.Open(filePath)
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		if lineIsWord(scanner.Text()) {
			out += 1
		}
	}

	if err := scanner.Err(); err != nil {
		pterm.Error.Printf("error reading wordlist: %s\n", err)
	}

	return out
}
