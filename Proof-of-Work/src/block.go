package src

import (
	"crypto/sha256"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"bytes"

	"github.com/libp2p/go-libp2p/core/peer"
)

// twenty-three "0"
var DIFFICULTY = []byte{0, 0, 2}

type Block struct {
	Round    int    `json:"round"`
	PrevHash []byte `json:"previous_hash"`
	Sign     Sign   `json:"signature"`
	Data     string `json:"data"`
	Nonce    int    `json:"nonce"`
}

type Sign struct {
	PeerID peer.ID `json:"peer_id"`
	Sign   []byte  `json:"signature"`
}

func createSign() Sign {
	hash := sha256.Sum256([]byte(PEER_ID))
	sign, _ := ecdsa.SignASN1(rand.Reader, PRIV, hash[:])
	return Sign{PEER_ID, sign}
}

func newBlock(round int, prevHash []byte, data string) Block {
	return mineBlock(round, prevHash, data)
}

func mineBlock(round int, prevHash []byte, data string) Block {
	log.Printf("info: mining block...")
	block := Block{round, prevHash, createSign(), data, 0}

	for nonce := 0; ; nonce++ {
		if nonce%1000000 == 0 {
			log.Printf("info: nonce: %d", nonce)
		}
		block.Nonce = nonce
		j, _ := json.Marshal(block)
		if hash := sha256.Sum256(j); bytes.Compare(hash[:], DIFFICULTY) == -1 {
			log.Printf(
				"info: mined! nonce: %d, hash: %s",
				nonce,
				hex.EncodeToString(hash[:]))
			return block
		}
	}
}
