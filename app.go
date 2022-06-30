package main

import (
	"encoding/hex"
	"log"
	"strings"
	"time"
)

type App struct {
	blocks []Block
}

func NewApp() App {
	app := &App{blocks: make([]Block, 0)}
	app.genesis()
	return *app
}

func (app *App) genesis() {
	genesisBlock := Block{
		0,
		"0000f816a87f806bb0073dcf026a64fb40c946b5abee2573702828694d5b4c43",
		"genesis",
		time.Now().Unix(),
		"genesis!",
		2836,
	}
	app.blocks = append(app.blocks, genesisBlock)
}

func (app *App) tryAddBlock(block Block) {
	latestBlock := app.blocks[len(app.blocks)-1]
	if isBlockValid(&block, &latestBlock) {
		app.blocks = append(app.blocks, block)
	} else {
		log.Printf("error: could not add block - invalid")
	}
}

func isBlockValid(block *Block, previousBlock *Block) bool {
	if block.PreviousHash != previousBlock.Hash {
		log.Printf("warn: block with id: %d has wrong previous hash", block.Id)
		return false
	}
	hash, err := hex.DecodeString(block.Hash)
	if err != nil {
		log.Printf("warn: can decode from hex")
	}
	if !strings.HasPrefix(hashToBinaryRepresentation(hash), DIFFICULTY_PREFIX) {
		log.Printf("warn: block with id: %d has invalid difficulty", block.Id)
		return false
	}
	if block.Id != previousBlock.Id + 1 {
		log.Printf("warn: block with id: %d is not the next block after the latest: %d",
			block.Id, previousBlock.Id)
		return false
	}
	if hex.EncodeToString(calculateHash(
		block.Id,
		block.Timestamp,
		block.PreviousHash,
		block.Data,
		block.Nonce,
	)) != block.Hash {
		log.Printf("warn: block with id: %d has invalid hash", block.Id)
		return false
	}
	return true
}

func (app *App) isChainValid(chain *[]Block) bool {
	for i := 0; i < len(*chain); i++ {
		if i == 0 {
			continue
		}
		first := (*chain)[i-1]
		second := (*chain)[i]
		if !isBlockValid(&second, &first) {
			return false
		}
	}
	return true
}

// We always choose the longest valid chain
func (app *App) chooseChain(local []Block, remote []Block) []Block {
	isLocalValid := app.isChainValid(&local)
	isRemotevalid := app.isChainValid(&remote)

	if isLocalValid && isRemotevalid {
		if len(local) >= len(remote) {
			return local
		} else {
			return remote
		}
	} else if isRemotevalid && !isLocalValid {
		return remote
	} else if !isRemotevalid && isLocalValid {
		return local
	} else {
		panic("local and remote chains are both invalid")
	}
}
