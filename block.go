package main

import (
	"fmt"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"strings"
	"time"
)

const DIFFICULTY_PREFIX string = "00"

type Block struct {
	Id           uint64 `json:"id"`
	Hash         string `json:"hash"`
	PreviousHash string `json:"previous_hash"`
	Timestamp    int64  `json:"timestamp"`
	Data         string `json:"data"`
	Nonce        uint64 `json:"nonce"`
}

func NewBlock(id uint64, previousHash string, data string) Block {
	now := time.Now()
	nonce, hash := mineBlock(id, now.Unix(), previousHash, data)
	return Block{
		id,
		hash,
		previousHash,
		now.Unix(),
		data,
		nonce,
	}
}

func calculateHash(id uint64, timestamp int64, previousHash string, data string, nonce uint64) []byte {
	block, _ := json.Marshal(Block{
		Id:           id,
		PreviousHash: previousHash,
		Timestamp:    timestamp,
		Data:         data,
		Nonce:        nonce,
	})
	hash := sha256.Sum256([]byte(block))
	return hash[:]
}

func mineBlock(id uint64, timestamp int64, previousHash string, data string) (uint64, string) {
	log.Printf("info: mining block...")
	var nonce uint64 = 0
	var hash []byte

	for {
		if nonce%100000 == 0 {
			log.Printf("info: nonce: %d", nonce)
		}
		hash = calculateHash(id, timestamp, previousHash, data, nonce)
		binaryHash := hashToBinaryRepresentation(hash)
		if strings.HasPrefix(binaryHash, DIFFICULTY_PREFIX) {
			log.Printf(
				"info: mined! nonce: %d, hash: %s, binary hash: %s",
				nonce,
				hex.EncodeToString(hash),
				binaryHash)
			break
		}
		nonce++
	}
	return nonce, hex.EncodeToString(hash)
}

func hashToBinaryRepresentation(hash []byte) string {
	res := ""
	for _, c := range hash {
		res += fmt.Sprintf("%b", c)
	}
	return res
}
