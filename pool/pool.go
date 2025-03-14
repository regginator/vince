package pool

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
)

// Simple proxy pool provider package

type Pool struct {
	Proxies []string

	index       int
	accessMutex sync.Mutex
}

// Intialize a new pool with a proxy list file
func New(filePath string) (*Pool, error) {
	pool := Pool{}

	{
		f, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}

		proxies := []string{}

		scanner := bufio.NewScanner(f)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			} else if _, err := url.Parse(line); err != nil {
				continue
			}

			proxies = append(proxies, line)
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}

		pool.Proxies = proxies
	}

	return &pool, nil
}

// Request a proxy from the pool
func (pool *Pool) Get() (string, error) {
	pool.accessMutex.Lock()
	defer pool.accessMutex.Unlock()

	poolLen := len(pool.Proxies)
	if poolLen == 0 {
		return "", fmt.Errorf("no proxies available in the pool")
	}

	proxy := pool.Proxies[pool.index]
	if pool.index == poolLen-1 {
		pool.index = 0
	} else {
		pool.index++
	}

	return proxy, nil
}
