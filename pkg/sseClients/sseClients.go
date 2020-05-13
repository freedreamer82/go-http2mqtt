package sseClients

import (
	"fmt"

	uuid "github.com/satori/go.uuid"
)

type SseCLients struct {
	clients    map[*SseClient]bool
}

type SseClientData map[string]interface{}

type SseClient struct {
	Id   string
	Data SseClientData
}


func (m *SseCLients)RegisterNewCLient() *SseClient{

	var err error
	u1 := uuid.Must(uuid.NewV4(), err)

	cl := SseClient{Id:u1.String(),Data:nil}



	return  &cl
}

func (m *SseCLients)GetClientsList()  map[*SseClient]bool{

	return m.clients
}

func (m *SseCLients)RegisterNewCLientWithData( data SseClientData ) *SseClient{
	var err error
	u1 := uuid.Must(uuid.NewV4(), err)

	cl := SseClient{Id:u1.String(),Data:data}

	m.clients[&cl] = true

	return  &cl
}

func New() *SseCLients {

	manager := new(SseCLients)

	manager.clients = make(map[*SseClient]bool)

	return manager
}

func (manager *SseCLients) RemoveCLient(conn * SseClient) {
	if _, ok := manager.clients[conn]; ok {
		fmt.Println("removing" + conn.Id)
		delete(manager.clients, conn)
	}
}
