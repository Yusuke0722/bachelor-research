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
	MESSAGES = []Message{}
	MESSAGES2 = []Message23{}
	MESSAGES3 = []Message23{}
	MESSAGES4 = []Message4{}
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
	FromPeerId peer.ID `json:"from_peer_id"`
}

type MessageRequest struct {
	Message    Message `json:"message"`
	FromPeerId peer.ID `json:"from_peer_id"`
}

type Message23Request struct {
	Message    Message23 `json:"message"`
	FromPeerId peer.ID   `json:"from_peer_id"`
}

type Message4Request struct {
	Message    Message4 `json:"message"`
	FromPeerId peer.ID  `json:"from_peer_id"`
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
			respMessage   MessageRequest
			respMessage23 Message23Request
			respMessage4  Message4Request
		)
		if json.Unmarshal(msg, &respChain);
				respChain.Receiver != "" && respChain.Receiver == PEER_ID {
			log.Printf("info: Response from %s", respChain.Sender)
			app.Blocks = app.chooseChain(
				app.Blocks, respChain.Blocks)
		} else if json.Unmarshal(msg, &respBlock);
				respBlock.Block.SSeed.PeerID != "" &&
				respBlock.Block.SSeed.PeerID == respBlock.FromPeerId {
			log.Printf("info: received new block from %s", respBlock.FromPeerId)
			app.tryAddBlock(respBlock.Block)
		} else if json.Unmarshal(msg, &respMessage23);
				respMessage23.Message.PeerID != "" &&
				respMessage23.Message.PeerID == respMessage23.FromPeerId &&
				respMessage23.Message.Sign.Seed.Step == 2 {
			log.Printf("info: received new message2 from %s", respMessage23.FromPeerId)
			mutex.Lock()
			MESSAGES2 = append(MESSAGES2, respMessage23.Message)
			mutex.Unlock()
		} else if respMessage23.Message.PeerID != "" &&
				respMessage23.Message.PeerID == respMessage23.FromPeerId &&
				respMessage23.Message.Sign.Seed.Step == 3 {
			log.Printf("info: received new message3 from %s", respMessage23.FromPeerId)
			mutex.Lock()
			MESSAGES3 = append(MESSAGES3, respMessage23.Message)
			mutex.Unlock()
		} else if json.Unmarshal(msg, &respMessage4);
				respMessage4.Message.PeerID != "" &&
				respMessage4.Message.PeerID == respMessage4.FromPeerId {
			log.Printf("info: recieved new message%d from %s",
				respMessage4.Message.Sign.Seed.Step, respMessage4.FromPeerId)
			mutex.Lock()
			MESSAGES4 = append(MESSAGES4, respMessage4.Message)
			mutex.Unlock()
		} else if json.Unmarshal(msg, &respMessage);
				respMessage.Message.Sign.PeerID != "" &&
				respMessage.Message.Sign.PeerID == respMessage.FromPeerId {
			log.Printf("info: received new message from %s", respMessage.FromPeerId)
			mutex.Lock()
			MESSAGES = append(MESSAGES, respMessage.Message)
			mutex.Unlock()
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
		block, isCast := newBlock(
			latestBlock.Round+1,
			hash[:],
			app.Seed,
			data)
		if isCast {
			log.Printf("info: broadcast new block")
			Publish(BlockRequest{block, PEER_ID})
			app.tryAddBlock(block)
		} else {
			log.Printf("info: you are not a leader")
		}
	}
}
