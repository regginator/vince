package main

// Interface that different bruteforce modes (wordlist, raw) must implement
type IterProvider interface {
	GetPasswordCount() uint64 // Used primarily to show current progress status

	IterPasswords() func(func(string) bool) // Actual iterator function
}
