package sseClients

import (
	//	"fmt"

	"sync"

	uuid "github.com/satori/go.uuid"
)

type SseCLients struct {
	clients map[*SseClient]bool
	mutex   sync.Mutex
}

type SseClientData map[string]interface{}

type SseClient struct {
	Id   string
	Data SseClientData
}

func (m *SseCLients) RegisterNewCLient() *SseClient {

	m.mutex.Lock()
	defer m.mutex.Unlock()

	var err error
	u1 := uuid.Must(uuid.NewV4(), err)

	cl := SseClient{Id: u1.String(), Data: nil}

	return &cl
}

func (m *SseCLients) GetClientsList() map[*SseClient]bool {

	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.clients
}

func (m *SseCLients) RegisterNewCLientWithData(data SseClientData) *SseClient {

	m.mutex.Lock()
	defer m.mutex.Unlock()

	var err error
	u1 := uuid.Must(uuid.NewV4(), err)

	cl := SseClient{Id: u1.String(), Data: data}

	m.clients[&cl] = true

	return &cl
}

func New() *SseCLients {

	manager := new(SseCLients)

	manager.clients = make(map[*SseClient]bool)

	return manager
}

func (manager *SseCLients) SetDisconnected(conn *SseClient) {

	manager.mutex.Unlock()
	defer manager.mutex.Unlock()

	if _, ok := manager.clients[conn]; ok {
		manager.clients[conn] = false
	}
}

func (manager *SseCLients) RemoveCLient(conn *SseClient) {

	manager.mutex.Lock()

	if _, ok := manager.clients[conn]; ok {
		//	fmt.Println("removing" + conn.Id)
		delete(manager.clients, conn)
	}

	manager.mutex.Unlock()
}

func (m *SseCLients) PurgeDisconnected() {

	m.mutex.Lock()
	defer m.mutex.Unlock()

	cls := m.clients

	for cl, isConnected := range m.clients {
		if !isConnected {
			delete(cls, cl)
		}

	}

	m.clients = cls

}

func (m *SseCLients) FuncIterationForClients(fn func(client *SseClient, isConnected bool) bool) bool {

	m.mutex.Lock()
	defer m.mutex.Unlock()

	for ssecl, isConnected := range m.clients {
		if ok := fn(ssecl, isConnected); ok == false {
			return false
		}
	}

	return true
}
