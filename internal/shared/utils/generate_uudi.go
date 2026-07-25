package utils

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"time"
)

func GenerateObjectIDLike() string {
	b := make([]byte, 12)

	// 4 bytes timestamp (seconds)
	binary.BigEndian.PutUint32(b[0:4], uint32(time.Now().Unix()))

	// 8 bytes random
	_, err := rand.Read(b[4:])
	if err != nil {
		panic(err)
	}

	return hex.EncodeToString(b)
}
