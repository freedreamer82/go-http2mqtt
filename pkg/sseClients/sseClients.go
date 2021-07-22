package sseClients

import (
	"sync"

	uuid "github.com/satori/go.uuid"
)

type SseCLients struct {
	// clients map[*SseClient]bool
	clients sync.Map
}

type SseClientData map[string]interface{}

type SseClient struct {
	Id   string
	Data SseClientData
}

func (m *SseCLients) RegisterNewCLient() *SseClient {

	var err error
	u1 := uuid.Must(uuid.NewV4(), err)

	cl := SseClient{Id: u1.String(), Data: nil}

	return &cl
}

// func (m *SseCLients) GetClientsList() map[*SseClient]bool {

// 	return m.clients
// }

func (m *SseCLients) GetRegisteredClients() int {

	cnt := 0
	m.clients.Range(func(key, value interface{}) bool {
		cnt++
		return true
	})

	return cnt

}

func (m *SseCLients) RegisterNewCLientWithData(data SseClientData) *SseClient {

	var err error
	u1 := uuid.Must(uuid.NewV4(), err)

	cl := SseClient{Id: u1.String(), Data: data}

	//func (m *Map) Store(key, value interface{})
	// Key and value are interface types, which can store data of any type
	m.clients.Store(&cl, true)

	return &cl
}

func New() *SseCLients {

	manager := new(SseCLients)

	return manager
}

func (manager *SseCLients) SetDisconnected(conn *SseClient) {
	// func (m *Map) Load(key interface{}) (value interface{}, ok bool)
	// Key: key, value: if the data exists, the corresponding value will be returned; if not, nil will be returned. ok: indicates whether the value is found
	// Remember to process data assertions
	_, ok := manager.clients.Load(conn)
	if ok {
		manager.clients.Store(conn, false)
	}

}

func (manager *SseCLients) RemoveCLient(conn *SseClient) {

	manager.clients.Delete(conn)

}

func (m *SseCLients) PurgeDisconnected() {

	//func (m *Map) Range(f func(key, value interface{}) bool)
	// The callback function used by the user to process data. The parameters of the callback function are key, value and the return value is bool
	m.clients.Range(func(key, value interface{}) bool {
		if value == false {
			m.clients.Delete(key)
		}

		return true
	})

}

func (m *SseCLients) FuncIterationForClients(fn func(client *SseClient, isConnected bool) bool) bool {

	m.clients.Range(func(key, value interface{}) bool {
		if value == true {
			if ok := fn(key.(*SseClient), value.(bool)); ok == false {
				return false
			}
		}
		return true
	})

	return true
}
