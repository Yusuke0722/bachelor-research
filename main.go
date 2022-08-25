package main

import (
	"proof-of-work/src"

	"bufio"
	"context"
	"log"
	"os"
	"strings"
	"sync"

	"net/http"
	_ "net/http/pprof"

	"github.com/comail/colog"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

var (
	app = src.NewApp()
	mutex = &sync.Mutex{}
)

func main() {
	go func ()  {
		log.Print(http.ListenAndServe("localhost:6060", nil))
	}()

	colog.SetFormatter(&colog.StdFormatter{
		Colors: true,
		Flag:   log.Ldate | log.Ltime | log.Lshortfile,
	})
	colog.Register()

	host, err := libp2p.New(
		libp2p.Identity(src.KEYS.PrivKey()),
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
	)
	if err != nil {
		log.Printf("warn: can start")
	}
	log.Printf("info: Peer Id: %s", host.ID())

	host.SetStreamHandler("/p2p/1.0.0", handleStream)

	peerInfo := peer.AddrInfo{
		ID:    host.ID(),
		Addrs: host.Addrs(),
	}
	addrs, _ := peer.AddrInfoToP2pAddrs(&peerInfo)
	log.Printf("node address: %s", addrs[0])

	if len(os.Args) > 1 {
		log.Printf("info: sending init event")
		var addr ma.Multiaddr
		var pi *peer.AddrInfo
		for i := 1; i < len(os.Args); i++ {
			addr, _ = ma.NewMultiaddr(os.Args[i])
			pi, _ = peer.AddrInfoFromP2pAddr(addr)
			if err := host.Connect(context.Background(), *pi); err != nil {
				log.Printf("error: can connect")
			}
			if s, err := host.NewStream(
					context.Background(),
					pi.ID,
					"/p2p/1.0.0"); err != nil {
				log.Printf("error: can create stream")
			} else {
				rw := bufio.NewReadWriter(
					bufio.NewReader(s), bufio.NewWriter(s))
				mutex.Lock()
				src.READWRITERS = append(src.READWRITERS, rw)
				mutex.Unlock()

				go app.InjectEvent(rw)
			}
		}
		peers := host.Peerstore().Peers()
		log.Printf("info: connected nodes: %d", peers.Len()-1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	var cmd string
	for {
		scanner.Scan()
		cmd = scanner.Text()
		if cmd == "ls p" {
			src.HandlePrintPeers(host)
		} else if cmd == "ls c" {
			src.HandlePrintChains(&app)
		} else if strings.HasPrefix(cmd, "create b ") {
			src.HandleCreateBlock(cmd, &app)
		} else {
			log.Printf("error: unknown command")
		}
	}
}

func handleStream(s network.Stream) {
	log.Printf("info: get new Stream %s", s.ID())
	rw := bufio.NewReadWriter(
		bufio.NewReader(s), bufio.NewWriter(s))
	mutex.Lock()
	src.READWRITERS = append(src.READWRITERS, rw)
	mutex.Unlock()
	log.Printf("info: sending local chain to %s", s.Conn().RemotePeer())
	src.Publish(src.ChainResponse{
		Blocks: app.Blocks,
		Sender: src.PEER_ID,
		Receiver: s.Conn().RemotePeer(),
	})

	go app.InjectEvent(rw)
}
