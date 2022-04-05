package clients

import (
	"sync"

	uuid "github.com/satori/go.uuid"
)

type Clients struct {
	// clients map[*Client]bool
	clients sync.Map
}

type ClientData map[string]interface{}
type ClientType uint8

const (
	SseClient ClientType = 1
	WsClient  ClientType = 2
)

func (ct ClientType) IsValid() bool {
	switch ct {
	case SseClient, WsClient:
		return true
	}
	return false
}

type Client struct {
	Id         string
	Data       ClientData
	clientType ClientType
}

func (m *Clients) RegisterNewClient(clientType ClientType) *Client {

	var err error
	u1 := uuid.Must(uuid.NewV4(), err)

	if !clientType.IsValid() {
		panic("client type not handled")
	}

	cl := Client{Id: u1.String(), Data: nil, clientType: clientType}

	return &cl
}

// func (m *Clients) GetClientsList() map[*Client]bool {

// 	return m.clients
// }

func (m *Clients) GetRegisteredClients() int {

	cnt := 0
	m.clients.Range(func(key, value interface{}) bool {
		cnt++
		return true
	})

	return cnt

}

func (m *Clients) RegisterNewClientWithData(clientType ClientType, data ClientData) *Client {

	var err error
	u1 := uuid.Must(uuid.NewV4(), err)

	if !clientType.IsValid() {
		panic("client type not handled")
	}

	cl := Client{Id: u1.String(), Data: data, clientType: clientType}

	//func (m *Map) Store(key, value interface{})
	// Key and value are interface types, which can store data of any type
	m.clients.Store(&cl, true)

	return &cl
}

func New() *Clients {

	manager := new(Clients)

	return manager
}

func (manager *Clients) SetDisconnected(conn *Client) {
	// func (m *Map) Load(key interface{}) (value interface{}, ok bool)
	// Key: key, value: if the data exists, the corresponding value will be returned; if not, nil will be returned. ok: indicates whether the value is found
	// Remember to process data assertions
	_, ok := manager.clients.Load(conn)
	if ok {
		manager.clients.Store(conn, false)
	}

}

func (manager *Clients) RemoveClient(conn *Client) {

	manager.clients.Delete(conn)

}

func (m *Clients) PurgeDisconnected() {

	//func (m *Map) Range(f func(key, value interface{}) bool)
	// The callback function used by the user to process data. The parameters of the callback function are key, value and the return value is bool
	m.clients.Range(func(key, value interface{}) bool {
		if value == false {
			m.clients.Delete(key)
		}

		return true
	})

}

func (m *Clients) FuncIterationForClients(fn func(client *Client, isConnected bool) bool) bool {

	m.clients.Range(func(key, value interface{}) bool {
		if value == true {
			if ok := fn(key.(*Client), value.(bool)); ok == false {
				return false
			}
		}
		return true
	})

	return true
}
