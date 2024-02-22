package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
    id string
    hub *Hub
    conn *websocket.Conn
    send chan []byte
}

const (
    // Time allowed to read the next pong message from the peer.
    pongWait = 60 * time.Second
    // Maximum message size allowed frorm peer.
    maxMessageSize = 512
    // send pings to peer wit hthis period, must be less than pongWait.
    pingPeriod = (pongWait * 9) / 10
    // time allowed to write a message to the peer.
    writeWait = 10 * time.Second
)

var upgrader = websocket.Upgrader{
    ReadBufferSize: 1024,
    WriteBufferSize: 1024,
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
    // upgrade the connection
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return
    }
    // create client
    id := uuid.New()
    client := &Client{id: id.String(), hub: hub, conn: conn, send: make(chan []byte, 256)}
    client.hub.register <- client

    // listening
    go client.writePump()
    go client.readPump()
}

func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()

    c.conn.SetReadLimit(maxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })

    for {
        _, text, err := c.conn.ReadMessage()
        log.Printf("value %v", string(text))
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("error: %v", err)
            }
            break
        }

        msg := &Message{}
        reader := bytes.NewReader(text)
        decoder := json.NewDecoder(reader)
        err = decoder.Decode(msg)
        if err != nil {
            log.Printf("error: %v", err)
        }

        c.hub.broadcast <- &Message{ClientID: c.id, Text: msg.Text}
    }
}

func (c *Client) writePump() {
    ticker := time.NewTicker(pingPeriod)
    defer func () {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case msg, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            w, err := c.conn.NextWriter(websocket.TextMessage)
            if err != nil {
                return
            }
            w.Write(msg)
            n := len(c.send)
            for i := 0; i < n; i++ {
                w.Write(msg)
            }
            if err := w.Close(); err != nil {
                return
            }
        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}
