package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"bytes"

	"log"

	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-pubsub"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

type Keys struct {
	privKey crypto.PrivKey
	pubKey  crypto.PubKey
}

var ( // immutable
	ctx                = context.Background()
	src                io.Reader
	privKey, pubKey, _ = crypto.GenerateEd25519Key(src)
	KEYS               = Keys{privKey, pubKey}
	PEER_ID, _         = peer.IDFromPublicKey(KEYS.pubKey)
	CHAIN_TOPIC        *pubsub.Topic
	BLOCK_TOPIC        *pubsub.Topic
	CHAIN_SUB          *pubsub.Subscription
	BLOCK_SUB	       *pubsub.Subscription
)

type ChainResponse struct {
	Blocks   []Block `json:"blocks"`
	Receiver string  `json:"receiver"`
}

type LocalChainRequest struct {
	FromPeerId string `json:"from_peer_id"`
}

type AppBehavior struct {
	pubsub       *pubsub.PubSub
	kdht           *dht.IpfsDHT
	responseSender chan ChainResponse
	initSender     chan bool
	app            App
}

func (keys Keys) PrivKey() crypto.PrivKey {
	return keys.privKey
}

func NewAppBehavior(
	host host.Host,
	app App,
	responseSender chan ChainResponse,
	initSender chan bool) AppBehavior {
	behavior := make(chan AppBehavior)
	go func() {
		kdht, err := dht.New(ctx, host)
		if err != nil {
			log.Printf("warn: has to exist")
		}
		if err = kdht.Bootstrap(ctx); err != nil {
			log.Printf("warn: has to exist")
		}
		pubsub, _ := pubsub.NewFloodSub(ctx, host)
		tmp := AppBehavior{
			pubsub,
			kdht,
			responseSender,
			initSender,
			app,
		}
		CHAIN_TOPIC, _ = tmp.pubsub.Join("chains")
		BLOCK_TOPIC, _ = tmp.pubsub.Join("blocks")
		CHAIN_SUB, _ = CHAIN_TOPIC.Subscribe()
		BLOCK_SUB, _ = BLOCK_TOPIC.Subscribe()
		behavior <- tmp
	}()
	return <-behavior
}

func (appBehavior *AppBehavior) injectEvent(sub *pubsub.Subscription) {
	msg, err := sub.Next(ctx)
	if err != nil {
		return
	}
	var (
		respChain ChainResponse
		respLocal LocalChainRequest
		block Block
	)
	if err := json.Unmarshal(msg.GetData(), &respChain); err == nil {
		if respChain.Receiver == PEER_ID.String() {
			log.Printf("info: Response from %s", msg.ReceivedFrom.String())
			s := ""
			for _, block := range respChain.Blocks {
				s = fmt.Sprintf("%s\n%+v", s, block)
			}
			log.Printf("info: %s", s)

			appBehavior.app.blocks = appBehavior.app.chooseChain(
				appBehavior.app.blocks, respChain.Blocks)
			return
		}
	}
	if err := json.Unmarshal(msg.GetData(), &respLocal); err == nil {
		if PEER_ID.String() == respLocal.FromPeerId {
			log.Printf("info: sending local chain to %s", msg.ReceivedFrom.String())
			appBehavior.responseSender <- ChainResponse{
				appBehavior.app.blocks,
				msg.ReceivedFrom.String(),
			}
			return
		}
	}
	if err := json.Unmarshal(msg.GetData(), &block); err == nil {
		if strings.HasPrefix(block.Hash, DIFFICULTY_PREFIX) {
			log.Printf("info: received new block from %s", msg.ReceivedFrom.String())
			appBehavior.app.tryAddBlock(block)
		}
	}
}

func GetListPeers(host host.Host) peer.IDSlice {
	log.Printf("info: Discovered Peers:")
	nodes := host.Peerstore().Peers()
	return nodes[1:]
}

func HandlePrintPeers(host host.Host) {
	peers := GetListPeers(host)
	for _, p := range peers {
		log.Printf("info: %s", p)
	}
}

func HandlePrintChains(appBehavior *AppBehavior) {
	log.Printf("info: Local Blockchain:")
	j, err := json.Marshal(appBehavior.app.blocks)
    if err != nil {
        log.Printf("warn: can jsonify blocks")
    }

	var out bytes.Buffer
	if err := json.Indent(&out, j, "", "  "); err != nil {
		log.Printf("warn: can indent json")
	}
	log.Printf("info: %s", out.String())
}

func HandleCreateBlock(cmd string, appBehavior *AppBehavior) {
	data := strings.TrimPrefix(cmd, "create b ")
	if data == "" {
		log.Printf("error: invalid command")
	} else {
		log.Printf("info: %s", data)
		latestBlock := appBehavior.app.blocks[len(appBehavior.app.blocks)-1]
		block := NewBlock(latestBlock.Id + 1, latestBlock.Hash, data)
		j, err := json.Marshal(block)
		if err != nil {
			log.Printf("warn: can jsonify request")
		}
		BLOCK_TOPIC.Publish(ctx, j)
		log.Printf("info: broadcasting new block")
	}
}
