package main

import (
	log "github.com/cihub/seelog"
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"os"
	"strconv"
	"time"
	"github.com/davecgh/go-spew/spew"
	"github.com/joho/godotenv"
)

type Block struct {
	Index     int;
	Timestamp string;
	BPM       int;
	Hash      string;
	PrevHash  string;
}

var BlockChain [] Block;
//创建块时计算hash值的函数
func calculateHash(block Block) string {
	record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PrevHash
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

//创建块的函数
func gennerateBlock(oldBlock Block, BPM int) (Block, error) {
	var newBlock Block
	t := time.Now()
	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.BPM = BPM
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Hash = calculateHash(newBlock)
	return newBlock, nil

}

//验证数据块的函数
func isBlockValid(newBlock, oldBlock Block) bool {

	if oldBlock.Index+1 != newBlock.Index {
		return false
	}
	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}
	if calculateHash(newBlock) != newBlock.Hash {
		return false
	}

	return true

}

//确保各个节点都以最长的链为准
func replaceChain(newBlock [] Block) {
	if len(newBlock) > len(BlockChain) {
		BlockChain = newBlock
	}
}

//网络通信
//接着我们来建立各个节点间的网络，用来传递块、同步链状态等。
//我们先来声明一个全局变量 bcServer ，以 channel（译者注：channel 类似其他语言中的 Queue,代码中声明的是一个 Block 数组的 channel）的形式来接受块。
//

var bcServer chan []Block

func main() {

	//	读取配置文件加载端口
	err := godotenv.Load()
	if err != nil {
		log.Error(err)
	}
	bcServer = make(chan []Block)
	t := time.Now()
	genesisBlock := Block{0, t.String(), 0, "", ""}
	spew.Dump(genesisBlock)
	BlockChain = append(BlockChain, genesisBlock)

	server, err := net.Listen("tcp", ":"+os.Getenv("ADDR"))
	if err != nil {
		log.Error(err)
	}
	defer server.Close()
	for {
		conn, err := server.Accept()
		if err != nil {
			log.Error(err)
		}
		go handleConn(conn)
	}

}

/**
客户端通过 stdin 输入 BPM

以 BPM 的值来创建块，这里会用到前面的函数：generateBlock，isBlockValid，和 replaceChain

将新的链放在 channel 中，并广播到整个网络
 */
func handleConn(conn net.Conn) {
	io.WriteString(conn, "Enter a new BPM:")
	scanner := bufio.NewScanner(conn)
	go func() {

		for scanner.Scan() {
			bpm, err := strconv.Atoi(scanner.Text())
			if err != nil {
				log.Error("%v not a number: %v", scanner.Text(), err)
				continue
			}
			newBlock, err := gennerateBlock(BlockChain[len(BlockChain)-1], bpm)
			if err != nil {
				log.Error(err)
				continue
			}
			if isBlockValid(newBlock, BlockChain[len(BlockChain)-1]) {
				newBlock := append(BlockChain, newBlock)
				replaceChain(newBlock)
			}
			bcServer <- BlockChain
			io.WriteString(conn, "\n Enter a new BPM:")
		}
	}()

	defer conn.Close()

	go func() {
		for {
			time.Sleep(30 * time.Second)
			output, err := json.Marshal(BlockChain)
			if err != nil {
				log.Error(err)
			}
			io.WriteString(conn, string(output))
		}
	}()

	for _ = range bcServer {
		spew.Dump(BlockChain)
	}

}
