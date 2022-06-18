// websockets.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Spirit struct {
	X uint `json:"x"`
	Y uint `json:"y"`
}
type UserCommand struct {
	move uint8
}

var connToSpiritIndex map[*websocket.Conn]int = make(map[*websocket.Conn]int)
var userCommandPool map[*websocket.Conn]*UserCommand = make(map[*websocket.Conn]*UserCommand)

var spiritPool []*Spirit
var connPool []*websocket.Conn
var connPoolLock sync.Mutex

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func gameLoop() {
	for true {
		for conn, userCommand := range userCommandPool {
			if (userCommand.move & 0x1) != 0 {
				(spiritPool[connToSpiritIndex[conn]].Y) -= 1
			}
			if (userCommand.move & 0x2) != 0 {
				(spiritPool[connToSpiritIndex[conn]].Y) += 1
			}
			if (userCommand.move & 0x4) != 0 {
				(spiritPool[connToSpiritIndex[conn]].X) -= 1
			}
			if (userCommand.move & 0x8) != 0 {
				(spiritPool[connToSpiritIndex[conn]].X) += 1
			}
		}
		jsonSpiritPool, err := json.Marshal(spiritPool)
		if err != nil {
			log.Println(err)
			return
		}
		//fmt.Printf("%s", jsonSpiritPool)
		for _, conn := range connPool {
			//conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("hello")))
			conn.WriteMessage(websocket.TextMessage, jsonSpiritPool)
		}
		time.Sleep(time.Second / 60)
	}
}

func socketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	connPoolLock.Lock()
	connPool = append(connPool, conn)
	userCommandPool[conn] = &UserCommand{0}
	spiritPool = append(spiritPool, &Spirit{0, 0})
	connToSpiritIndex[conn] = len(spiritPool) - 1
	connPoolLock.Unlock()

	defer conn.Close()
	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		cmdCode, err := strconv.Atoi(string(message))
		if err != nil {
			log.Println("Atoi:", err)
			break
		}
		//w0001 s0010 a0100 d1000
		switch cmdCode {
		case 87: //'W'
			userCommandPool[conn].move ^= 0x1
			break
		case 83: //'S'
			userCommandPool[conn].move ^= 0x2
			break
		case 65: //'A'
			userCommandPool[conn].move ^= 0x4
			break
		case 68: //'D'
			userCommandPool[conn].move ^= 0x8
			break
		}

		fmt.Printf("recv: %c %p %d \n", message, conn, userCommandPool[conn].move)
		err = nil
		for _, connection := range connPool {
			if err == nil {
				err = connection.WriteMessage(mt, message)
			}
		}

		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func main() {
	http.HandleFunc("/echo", socketHandler)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "websockets.html")
	})

	go gameLoop()
	http.ListenAndServe(":5555", nil)
}
