package application

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type RandomIDs struct{}

func (RandomIDs) NewID(prefix string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return prefix + "-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	return prefix + "-" + hex.EncodeToString(b)
}
