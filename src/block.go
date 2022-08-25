package src

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

const lambda, Lambda, maxStep = 10, 60, 180

type Block struct {
	Round    int    `json:"round"`
	PrevHash []byte `json:"previous_hash"`
	SSeed    SSeed  `json:"signature"`
	Data     string `json:"data"`
}

type Seed struct {
	Round int    `json:"round"`
	Step  int    `json:"step"`
	Seed  string `json:"seed"`
}

type SSeed struct {
	PeerID peer.ID `json:"peer_id"`
	Seed   string  `json:"seed"`
	Sign   []byte  `json:"signature"`
}

type Sign struct {
	PeerID peer.ID `json:"peer_id"`
	Seed   Seed    `json:"seed"`
	Sign   []byte  `json:"signature"`
}

type Message struct {
	Block     Block  `json:"block"`
	Esig      []byte `json:"ephemeral_signature"`
	Sign      SSeed  `json:"signature"`
}

type Value struct {
	HashBlock []byte  `json:"hashblock"`
	Leader    peer.ID `json:"leader"`
}

type Message23 struct {
	PeerID     peer.ID `json:"peer_id"`
	Value      Value   `json:"value"`
	ValueESign []byte  `json:"value_sign"`
	Sign       Sign    `json:"signature"`
}

type Message4 struct {
	PeerID     peer.ID `json:"peer_id"`
	Bit        int     `json:"bit"`
	BESign     []byte  `json:"b_sign"`
	Value      Value   `json:"value"`
	ValueESign []byte  `json:"value_sign"`
	Sign       Sign    `json:"signature"`
}

func createSSeed(seed string) SSeed {
	hash := sha256.Sum256([]byte(seed))
	sign, _ := ecdsa.SignASN1(rand.Reader, PRIV, hash[:])
	return SSeed{PEER_ID, seed, sign}
}

func createSign(round int, step int, seed string) Sign {
	s := Seed{round, step, seed}
	j, _ := json.Marshal(s)
	hash := sha256.Sum256(j)
	sign, _ := ecdsa.SignASN1(rand.Reader, PRIV, hash[:])
	return Sign{PEER_ID, s, sign}
}

func newBlock(round int, prevHash []byte, seed string, data string) (Block, bool) {
	block := Block{
		round,
		prevHash,
		createSSeed(seed),
		data,
	}
	step1(block)
	tH := step2(round, seed)
	step3(round, seed, tH)
	value := step4(round, seed, tH)
	newBlockValue := Value{Leader: "nil"}
	found := false
	for step := 5; step < maxStep; step++ {
		if newBlockValue0, isFine0 := isFinalized0(round, step, tH); isFine0 {
			newBlockValue = newBlockValue0
			found = true
			break
		} else if newBlockValue1, isFine1 := isFinalized1(round, step, tH); isFine1 {
			newBlockValue = newBlockValue1
			found = true
			break
		}
		switch step % 3 {
		case 2:
			step5(round, step, seed, value, tH)
		case 0:
			step6(round, step, seed, value, tH)
		case 1:
			step7(round, step, seed, value, tH)
		}
	}

	MESSAGES, MESSAGES2, MESSAGES3, MESSAGES4 = []Message{}, []Message23{}, []Message23{}, []Message4{}

	if found && PEER_ID == newBlockValue.Leader {
		return block, true
	}
	return Block{}, false // invalid block
}

func step1(block Block) {
	log.Printf("info: proposing block...")
	j, _ := json.Marshal(block)
	hash := sha256.Sum256(j)
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	esign, _ := ecdsa.SignASN1(rand.Reader, priv, hash[:])
	m := Message{block, esign, block.SSeed}
	mutex.Lock()
	MESSAGES = append(MESSAGES, m)
	mutex.Unlock()
	Publish(MessageRequest{m, PEER_ID})
}

func findLeader(round int) Message {
	var lead [32]byte
	num := -1
	for i, message := range MESSAGES {
		if message.Block.Round == round {
			j, _ := json.Marshal(message)
			if num < 0 {
				num, lead = i, sha256.Sum256(j)
			} else if lead2 := sha256.Sum256(j);
					bytes.Compare(lead[:], lead2[:]) == 1 {
				num, lead = i, lead2
			}
		}
	}
	return MESSAGES[num]
}

func sendMessage2(value Value, round int, seed string) {
	v, _ := json.Marshal(value)
	h := sha256.Sum256(v)
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	esign, _ := ecdsa.SignASN1(rand.Reader, priv, h[:])
	sign := createSign(round, 2, seed)
	m := Message23{PEER_ID, value, esign, sign}
	mutex.Lock()
	MESSAGES2 = append(MESSAGES2, m)
	mutex.Unlock()
	Publish(Message23Request{m, PEER_ID})
}

func sendMessage3(value Value, round int, seed string) {
	v, _ := json.Marshal(value)
	h := sha256.Sum256(v)
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	esign, _ := ecdsa.SignASN1(rand.Reader, priv, h[:])
	sign := createSign(round, 3, seed)
	m := Message23{PEER_ID, value, esign, sign}
	mutex.Lock()
	MESSAGES3 = append(MESSAGES3, m)
	mutex.Unlock()
	Publish(Message23Request{m, PEER_ID})
}

func calcTH(round int) int {
	num := 0.0
	for _, m := range MESSAGES {
		if m.Block.Round == round { num++ }
	}
	return int(0.69 * num)
}

func step2(round int, seed string) int {
	log.Printf("info: step2...")
	c1, c2 := make(chan Value), make(chan Value)
	go func() {
		time.Sleep((lambda+Lambda) * time.Second)
		c1 <- Value{Leader: "nil"}
	}()

	go func() {
		time.Sleep(2*lambda * time.Second)
		lead := findLeader(round)
		j, _ := json.Marshal(lead.Block)
		hash := sha256.Sum256(j)
		c2 <- Value{hash[:], lead.Sign.PeerID}
	}()
	var value Value
	select {
	case value = <-c1:
		log.Printf("info: can't find value")
	case value = <-c2:
	}
	sendMessage2(value, round, seed)
	return calcTH(round)
}

func isLeader(value Value, values []Value, tH int) bool {
	count := 0
	for _, v := range values {
		if value.Leader == v.Leader { count++ }
		if count >= tH { return true }
	}
	return false
}

func findValue(messages []Message23, round int, tH int) (Value, bool) {
	var values []Value
	for _, m := range messages {
		if m.Sign.Seed.Round == round {
			values = append(values, m.Value)
			if isLeader(m.Value, values, tH) { return m.Value, true }
		}
	}
	return Value{}, false
}

func step3(round int, seed string, tH int) {
	log.Printf("info: step3...")
	c1, c2 := make(chan Value), make(chan Value)
	go func() {
		time.Sleep((3*lambda+Lambda) * time.Second)
		c1 <- Value{Leader: "nil"}
	}()

	go func() {
		var tmp Value
		var isFound bool
		for {
			if tmp, isFound = findValue(MESSAGES2, round, tH); isFound { break }
			time.Sleep(100 * time.Millisecond)
		}
		c2 <- tmp
	}()
	var value Value
	select {
	case value = <-c1:
		log.Printf("info: can't find value")
	case value = <-c2:
	}
	sendMessage3(value, round, seed)
}

func sendMessage4(bit int, value Value, round int, step int, seed string) {
	bj, _ := json.Marshal(bit)
	vj, _ := json.Marshal(value)
	bh, vh := sha256.Sum256(bj), sha256.Sum256(vj)
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	besign, _ := ecdsa.SignASN1(rand.Reader, priv, bh[:])
	vesign, _ := ecdsa.SignASN1(rand.Reader, priv, vh[:])
	sign := createSign(round, step, seed)
	m := Message4{PEER_ID, bit, besign, value, vesign, sign}
	mutex.Lock()
	MESSAGES4 = append(MESSAGES4, m)
	mutex.Unlock()
	Publish(Message4Request{m, PEER_ID})
}

func step4(round int, seed string, tH int) Value {
	log.Printf("info: step4...")
	c1, c2 := make(chan Value), make(chan Value)
	var g int
	go func() {
		time.Sleep(2*lambda * time.Second)
		tmp, isFound := findValue(MESSAGES3, round, tH/2)
		if isFound {
			if tmp.Leader != "nil" {g = 1} else {g = 0}
		}
		c1 <- tmp
	}()

	go func() {
		var tmp Value
		var isFound bool
		for {
			if tmp, isFound = findValue(MESSAGES3, round, tH); isFound { break }
			time.Sleep(100 * time.Millisecond)
		}
		if tmp.Leader != "nil" {g = 2} else {g = 0}
		c2 <- tmp
	}()
	var bit int
	var value Value
	select {
	case value = <-c1:
		log.Printf("info: can't find value")
	case value = <-c2:
	}
	if g == 2 {bit = 0} else {bit = 1}
	sendMessage4(bit, value, round, 4, seed)
	return value
}

func isFinalized0(round int, step int, tH int) (Value, bool) {
	var valids []Message4
	var count int
	for _, m := range MESSAGES4 {
		s := m.Sign.Seed.Step + 1
		if (m.Sign.Seed.Round == round &&
				m.Value.Leader != "nil" &&
				5 <= s && s <= step &&
				s % 3 == 2 &&
				m.Bit == 0) {
			count = 0
			valids = append(valids, m)
			for _, v := range valids {
				if (v.Bit == 0 &&
						v.Sign.Seed.Step == s - 1 &&
						bytes.Equal(m.Value.HashBlock, v.Value.HashBlock)) {
					count++
				}
				if count >= tH {return m.Value, true}
			}
		}
	}
	return MESSAGES4[0].Value, false
}

func isFinalized1(round int, step int, tH int) (Value, bool) {
	var valids []Message4
	var count int
	for _, m := range MESSAGES4 {
		s := m.Sign.Seed.Step + 1
		if (m.Sign.Seed.Round == round &&
				6 <= s && s <= step &&
				s % 3 == 0 &&
				m.Bit == 1) {
			count = 0
			valids = append(valids, m)
			for _, v := range valids {
				if (v.Bit == 1 &&
						v.Sign.Seed.Step == s - 1 &&
						bytes.Equal(m.Value.HashBlock, v.Value.HashBlock)) {
					count++
				}
				if count >= tH {return m.Value, true}
			}
		}
	}
	return MESSAGES4[0].Value, false
}

func coinFlipped(round int, step int, tH int, bit int) bool {
	var valids []Message4
	var count int
	for _, m := range MESSAGES4 {
		s := m.Sign.Seed.Step + 1
		if (m.Sign.Seed.Round == round && step == s && m.Bit == bit) {
			count = 0
			valids = append(valids, m)
			for _, v := range valids {
				if (v.Bit == bit &&
						v.Sign.Seed.Step == s - 1 &&
						bytes.Equal(m.Value.HashBlock, v.Value.HashBlock)) {
					count++
				}
				if count >= tH {
					return true
				}
			}
		}
	}
	return false
}

func step5(round int, step int, seed string, value Value, tH int) {
	log.Printf("info: step%d...", step)
	bit1, bit2 := make(chan int), make(chan int)
	go func() {
		time.Sleep(2*lambda * time.Second)
		bit1 <- 0
	}()
	go func() {
		for {
			if coinFlipped(round, step, tH, 1) { break }
			time.Sleep(100 * time.Millisecond)
		}
		bit2 <- 1
	}()
	var bit int
	select {
	case bit = <-bit1:
		log.Printf("info: can't find value")
	case bit = <-bit2:
	}
	sendMessage4(bit, value, round, step, seed)
}

func step6(round int, step int, seed string, value Value, tH int) {
	log.Printf("info: step%d...", step)
	bit1, bit2 := make(chan int), make(chan int)
	go func() {
		time.Sleep(2*lambda * time.Second)
		bit1 <- 1
	}()
	go func() {
		for {
			if coinFlipped(round, step, tH, 0) { break }
			time.Sleep(100 * time.Millisecond)
		}
		bit2 <- 0
	}()
	var bit int
	select {
	case bit = <-bit1:
		log.Printf("info: can't find value")
	case bit = <-bit2:
	}
	sendMessage4(bit, value, round, step, seed)
}

func step7(round int, step int, seed string, value Value, tH int) {
	log.Printf("info: step%d...", step)
	bit1, bit2, bit3 := make(chan int), make(chan int), make(chan int)
	go func() {
		time.Sleep(2*lambda * time.Second)
		lead := findLeader(round)
		j, _ := json.Marshal([]interface{} {lead.Sign, round})
		hash := sha256.Sum256(j)
		bit1 <- int(hash[31]) % 2
	}()
	go func() {
		for {
			if coinFlipped(round, step, tH, 0) { break }
			time.Sleep(100 * time.Millisecond)
		}
		bit2 <- 0
	}()

	go func() {
		for {
			if coinFlipped(round, step, tH, 1) { break }
			time.Sleep(100 * time.Millisecond)
		}
		bit3 <- 1
	}()
	var bit int
	select {
	case bit = <-bit1:
		log.Printf("info: can't find value")
	case bit = <-bit2:
	case bit = <-bit3:
	}
	sendMessage4(bit, value, round, step, seed)
}
