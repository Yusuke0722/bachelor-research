package main

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"time"

	"log"

	"github.com/comail/colog"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	multiaddr "github.com/multiformats/go-multiaddr"
)

func main() {
	colog.SetFormatter(&colog.StdFormatter{
		Colors: true,
		Flag:   log.Ldate | log.Ltime | log.Lshortfile,
	})
	colog.Register()

	host, err := libp2p.New(
		libp2p.Identity(KEYS.PrivKey()),
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
	)
	if err != nil {
        log.Printf("warn: can start")
	}
	log.Printf("info: Peer Id: %s", host.ID())

	peerInfo := peer.AddrInfo{
		ID:    host.ID(),
		Addrs: host.Addrs(),
	}
	addrs, _ := peer.AddrInfoToP2pAddrs(&peerInfo)
	log.Printf("node address: %s", addrs[0])
	if len(os.Args) > 1 {
		var addr multiaddr.Multiaddr
		var pi *peer.AddrInfo
		for i := 1; i < len(os.Args); i++ {
			addr, _ = multiaddr.NewMultiaddr(os.Args[i])
			pi, _ = peer.AddrInfoFromP2pAddr(addr)
			host.Connect(ctx, *pi)
		}
	}

	behavior := NewAppBehavior(host, NewApp(), make(chan ChainResponse), make(chan bool))

	go func ()  {
		time.Sleep(time.Second * 1)
		log.Printf("info: sending init event")
		behavior.initSender <- true
	}()

	if <-behavior.initSender {
		peers := GetListPeers(host)

		log.Printf("info: connected nodes: %d", peers.Len())
		if peers.Len() > 0 {
			req := LocalChainRequest{FromPeerId: peers[peers.Len()-1].String()}

			json, err := json.Marshal(req)
			if err != nil {
				log.Printf("warn: can jsonify request")
			}
			CHAIN_TOPIC.Publish(ctx, json)
		}
	}

	go func () {
		for {
            json, err := json.Marshal(<-behavior.responseSender)
            if err != nil {
                log.Printf("warn: can jsonify response")
            }
            CHAIN_TOPIC.Publish(ctx, json)
        }
	}()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
        var cmd string
		for {
            scanner.Scan()
            cmd = scanner.Text()
            if cmd == "ls p" {
                HandlePrintPeers(host)
            } else if cmd == "ls c" {
				HandlePrintChains(&behavior)
            } else if strings.HasPrefix(cmd, "create b ") {
				HandleCreateBlock(cmd, &behavior)
            } else {
				log.Printf("error: unknown command")
            }
        }
	}()

	go func() {
		for {
			//time.Sleep(time.Millisecond * 10)
			behavior.injectEvent(CHAIN_SUB)
		}
	}()

	for {
		//time.Sleep(time.Millisecond * 10)
		behavior.injectEvent(BLOCK_SUB)
	}
}
