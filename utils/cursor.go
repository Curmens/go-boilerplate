package utils

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// EncodeCursor encodes a (createdAt, id) pair into an opaque base64 cursor string.
func EncodeCursor(createdAt time.Time, id uint) string {
	raw := fmt.Sprintf("%d:%d", createdAt.UnixNano(), id)
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor decodes a cursor string back into (createdAt, id).
func DecodeCursor(cursor string) (time.Time, uint, error) {
	b, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid cursor encoding")
	}
	parts := strings.SplitN(string(b), ":", 2)
	if len(parts) != 2 {
		return time.Time{}, 0, fmt.Errorf("invalid cursor format")
	}
	ns, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid cursor timestamp")
	}
	id, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid cursor id")
	}
	return time.Unix(0, ns).UTC(), uint(id), nil
}
