package src

import (
	"bufio"
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"log"
	"strings"
	"sync"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Keys struct {
	privKey crypto.PrivKey
	pubKey  crypto.PubKey
}

var ( // immutable
	PRIV, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	privKey, pubKey, _ = crypto.ECDSAKeyPairFromKey(PRIV)
	PEER_ID, _         = peer.IDFromPublicKey(pubKey)
	KEYS = Keys{privKey, pubKey}
	READWRITERS = []*bufio.ReadWriter{}
	mutex = &sync.Mutex{}
)

type ChainResponse struct {
	Blocks   []Block `json:"blocks"`
	Sender   peer.ID `json:"sender"`
	Receiver peer.ID `json:"receiver"`
}

type LocalChainRequest struct {
	ToPeerId   peer.ID `json:"to_peer_id"`
	FromPeerId peer.ID `json:"from_peer_id"`
}

type BlockRequest struct {
	Block      Block   `json:"block"`
	Nonce      int     `json:"nonce"`
	FromPeerId peer.ID `json:"from_peer_id"`
}

func (keys Keys) PrivKey() crypto.PrivKey {
	return keys.privKey
}

func Publish(data interface{}) {
	j, err := json.Marshal(data)
	if err != nil {
		log.Printf("warn: can jsonify")
	}
	for _, rw := range READWRITERS {
		rw.Write(append(j, '\n'))
		rw.Flush()
	}
}

func (app *App) InjectEvent(rw *bufio.ReadWriter) {
	for {
		msg, err := rw.ReadBytes('\n')
		if err != nil {
			log.Printf("warn: can read")
			return
		}
		var (
			respChain     ChainResponse
			respBlock     BlockRequest
		)
		if json.Unmarshal(msg, &respChain);
				respChain.Receiver != "" && respChain.Receiver == PEER_ID {
			log.Printf("info: Response from %s", respChain.Sender)
			app.Blocks = app.chooseChain(
				app.Blocks, respChain.Blocks)
		} else if json.Unmarshal(msg, &respBlock);
				respBlock.FromPeerId != "" &&
				respBlock.Block.Nonce == respBlock.Nonce {
			log.Printf("info: received new block from %s", respBlock.FromPeerId)
			app.tryAddBlock(respBlock.Block)
		}
	}
}

func getListPeers(host host.Host) peer.IDSlice {
	log.Printf("info: Discovered Peers:")
	nodes := host.Peerstore().Peers()
	return nodes[1:]
}

func HandlePrintPeers(host host.Host) {
	peers := getListPeers(host)
	for _, p := range peers {
		log.Printf("info: %s", p)
	}
}

func HandlePrintChains(app *App) {
	log.Printf("info: Local Blockchain:")
	j, err := json.Marshal(app.Blocks)
	if err != nil {
		log.Printf("warn: can jsonify blocks")
	}

	var out bytes.Buffer
	if err := json.Indent(&out, j, "", "  "); err != nil {
		log.Printf("warn: can indent json")
	}
	log.Printf("info: %s", out.String())
}

func HandleCreateBlock(cmd string, app *App) {
	data := strings.TrimPrefix(cmd, "create b ")
	if data == "" {
		log.Printf("error: invalid command")
	} else {
		latestBlock := app.Blocks[len(app.Blocks)-1]
		j, _ := json.Marshal(latestBlock)
		hash := sha256.Sum256(j)
		block := newBlock(latestBlock.Round+1, hash[:], data)
		log.Printf("info: broadcast new block")
		Publish(BlockRequest{block, block.Nonce, PEER_ID})
		app.tryAddBlock(block)
	}
}
