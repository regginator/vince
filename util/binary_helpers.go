package util

import (
	"encoding/binary"
	"io"
)

func ReadU32String(stream io.Reader, order binary.ByteOrder) (string, error) {
	var len uint32
	if err := binary.Read(stream, order, &len); err != nil {
		return "", err
	}

	str := make([]byte, len)
	if _, err := stream.Read(str); err != nil {
		return string(str), err
	}

	return string(str), nil
}
