package star

import (
	"bytes"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

const wsPeerAliveTTL = 60 * time.Second

var mSendMessage sync.Mutex

func processMessage(addressBook addressBook, handshakeSubscription *handshakeSubscription, message []byte) error {
	if bytes.Index(message, []byte(`["ws-peer",`)) == 0 {
		var m []string
		err := json.Unmarshal(message, &m)
		if err != nil {
			return err
		} else if len(m) < 2 {
			return errors.New("missing peer information")
		}
		return processWsPeerMessage(addressBook, m[1])
	} else if bytes.Index(message, []byte(`["ws-handshake",`)) == 0 {
		return processWsHandshakeMessage(handshakeSubscription, message[len(`["ws-handshake",`):len(message)-1])
	}
	return errors.New("tried to process unknown message")
}

func processWsPeerMessage(addressBook addressBook, message string) error {
	peerID, signalMultiaddr, err := extractPeerDestination(message)
	if err != nil {
		return err
	}

	addressBook.AddAddr(peerID, signalMultiaddr, wsPeerAliveTTL)
	return nil
}

func processWsHandshakeMessage(handshakeSubscription *handshakeSubscription, message []byte) error {
	var answer handshakeData
	err := json.Unmarshal(message, &answer)
	if err != nil {
		return err
	}
	handshakeSubscription.emit(answer)
	return nil
}

func extractPeerDestination(peerAddr string) (peer.ID, ma.Multiaddr, error) {
	peerMultiaddr, err := ma.NewMultiaddr(peerAddr)
	if err != nil {
		return "", nil, err
	}

	value, err := peerMultiaddr.ValueForProtocol(ma.P_IPFS)
	if err != nil {
		return "", nil, err
	}

	peerID, err := peer.IDB58Decode(value)
	if err != nil {
		return "", nil, err
	}

	ipfsMultiaddr, err := ma.NewMultiaddr("/ipfs/" + peerID.String())
	if err != nil {
		return "", nil, err
	}
	return peerID, peerMultiaddr.Decapsulate(ipfsMultiaddr), nil
}

func readMessage(connection *websocket.Conn) ([]byte, error) {
	_, message, err := connection.ReadMessage()
	if err != nil {
		return nil, err
	}

	i := bytes.IndexAny(message, "[{")
	if i < 0 {
		return nil, errors.New("message token not found")
	}
	return message[i:], nil
}

func sendMessage(connection *websocket.Conn, messageType string, messageBody interface{}) error {
	var buffer bytes.Buffer
	buffer.WriteString(messagePrefix)
	buffer.WriteString(`["`)
	buffer.WriteString(messageType)
	buffer.WriteByte('"')

	if messageBody != nil {
		b, err := json.Marshal(messageBody)
		if err != nil {
			return err
		}

		buffer.WriteByte(',')
		buffer.Write(b)
	}
	buffer.WriteByte(']')

	mSendMessage.Lock()
	defer mSendMessage.Unlock()
	return connection.WriteMessage(websocket.TextMessage, buffer.Bytes())
}

func readEmptyMessage(connection *websocket.Conn) error {
	_, message, err := connection.ReadMessage()
	if err != nil {
		return err
	}

	i := bytes.IndexByte(message, '{')
	if i > 0 {
		return errors.New("empty message expected")
	}
	return nil
}
