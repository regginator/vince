package util

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseNumRange(rangeStr string) (min int64, max int64, err error) {
	// Fine, I won't use regex in production
	rangeArr := strings.Split(rangeStr, "-")
	rangeLen := len(rangeArr)
	if rangeLen == 0 {
		return 0, 0, fmt.Errorf("number range is empty")
	} else if rangeLen > 2 {
		return 0, 0, fmt.Errorf("number range too large: expected max of (2) numbers, got (%d)", rangeLen)
	}

	min, err = strconv.ParseInt(rangeArr[0], 0, 64)
	if err != nil {
		return 0, 0, err
	} else if rangeLen == 1 {
		return min, min, nil
	}

	max, err = strconv.ParseInt(rangeArr[1], 0, 64)
	if err != nil {
		return 0, 0, err
	}

	return min, max, err
}
