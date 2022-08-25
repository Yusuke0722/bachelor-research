package src

import (
	"encoding/hex"
	"log"
	"bytes"
	"encoding/json"
	"crypto/sha256"
)

type App struct {
	Blocks []Block `json:"blocks"`
}

func NewApp() App {
	app := &App{make([]Block, 0)}
	app.genesis()
	return *app
}

func (app *App) genesis() {
	prevHash, _ := hex.DecodeString("0000f816a87f806bb0073dcf026a64fb40c946b5abee2573702828694d5b4c43")
	genesisBlock := Block{
		-1,
		prevHash,
		createSign(),
		"genesis!",
		0,
	}
	mutex.Lock()
	app.Blocks = append(app.Blocks, genesisBlock)
	mutex.Unlock()
}

func (app *App) tryAddBlock(block Block) {
	latestBlock := app.Blocks[len(app.Blocks)-1]
	if isBlockValid(block, latestBlock) {
		mutex.Lock()
		app.Blocks = append(app.Blocks, block)
		mutex.Unlock()
	} else {
		log.Printf("error: could not add block - invalid")
	}
}

func isBlockValid(block Block, previousBlock Block) bool {
	if block.Round != previousBlock.Round+1 {
		log.Printf("warn: block with id: %d is not the next block after the latest: %d",
			block.Round, previousBlock.Round)
		return false
	}
	j, _ := json.Marshal(previousBlock)
	if hash := sha256.Sum256(j); !bytes.Equal(block.PrevHash, hash[:]) {
		log.Printf("warn: block with id: %d has invalid hash", block.Round)
		return false
	}
	i, _ := json.Marshal(block)
	if hash := sha256.Sum256(i); !(bytes.Compare(hash[:], DIFFICULTY) == -1) {
		log.Printf("warn: block with id: %d has invalid difficulty", block.Round)
		return false
	}
	return true
}

func (app *App) isChainValid(chain *[]Block) bool {
	for i := 1; i < len(*chain); i++ {
		first := (*chain)[i-1]
		second := (*chain)[i]
		if !isBlockValid(second, first) {
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
		if len(local) > len(remote) {
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
