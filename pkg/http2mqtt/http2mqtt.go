package http2mqtt

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/freedreamer82/go-http2mqtt/pkg/clients"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	//"time"
)

const MaxClientIdLen = 14
const channelClientSize = 50

func (s *Http2Mqtt) SetGinAuth(user string, password string) {
	s.user = user
	s.password = password
}

// PublishMessage is the sample data transfer object for publish http route
type PublishMessage struct {
	Topic    string          `binding:"required" json:"topic"`
	Message  json.RawMessage `binding:"required" json:"data"`
	Qos      byte            `json:"qos"`
	Retained bool            `json:"retained"`
}

type SubScribeMessage struct {
	Topic string `binding:"required" json:"topic"`
	Qos   byte   `json:"qos"`
}

type Http2Mqtt struct {
	Router        *gin.Engine
	Group         *gin.RouterGroup
	MqttBrokerURL string
	mqttClient    MQTT.Client
	mqttOpts      *MQTT.ClientOptions
	user          string
	password      string
	clients       *clients.Clients
	subsMutex     sync.Mutex
	subs          []SubScribeMessage
	profileEnable bool
	prefixRestApi string
	streamEnabled bool
	streamChannel chan MQTT.Message
}

type WsHandler struct {
	id   string
	ws   *websocket.Conn
	send *chan MQTT.Message
}

func getRandomClientId() string {
	var characterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, MaxClientIdLen)
	for i := range b {
		b[i] = characterRunes[rand.Intn(len(characterRunes))]
	}
	id := "http2mqtt-" + string(b)
	log.Println("ID: ", id)
	return id
}

type Http2MqttOption func(*Http2Mqtt)

func WithOptionRouter(router *gin.Engine) Http2MqttOption {
	return func(h *Http2Mqtt) {
		h.Router = router
	}
}
func WithOptionRouterGroup(group *gin.RouterGroup) Http2MqttOption {
	return func(h *Http2Mqtt) {
		h.Group = group
	}
}

func WithOptionPrefix(prefix string) Http2MqttOption {
	return func(h *Http2Mqtt) {
		h.setRestPathPrefix(prefix)
	}
}

func New(mqttOpts *MQTT.ClientOptions, opts ...Http2MqttOption) *Http2Mqtt {

	rand.Seed(time.Now().UnixNano())

	h := Http2Mqtt{Router: nil, MqttBrokerURL: "", mqttOpts: nil, user: "", password: "", profileEnable: false, prefixRestApi: "", streamEnabled: true}

	for _, opt := range opts {
		// Call the option giving the instantiated
		// *House as the argument
		opt(&h)
	}

	if h.Router == nil {
		h.Router = gin.New()
	}
	h.mqttOpts = mqttOpts

	h.setupMQTT()
	h.setupGin()

	return &h
}

func (h *Http2Mqtt) GetMqttClient() *MQTT.Client {

	return &h.mqttClient
}

func (h *Http2Mqtt) EnableStream(status bool) {

	h.streamEnabled = status
}

func (h *Http2Mqtt) SetStreamChannel(ch chan MQTT.Message) {

	h.streamChannel = ch

}

func (h *Http2Mqtt) GetStreamChannel() chan MQTT.Message {

	return h.streamChannel
}

func (h *Http2Mqtt) GetSubscriptions() []SubScribeMessage {

	return h.subs
}

func (h *Http2Mqtt) Run(addrHttp string) {

	defer func(add string) {
		go h.Router.Run(add)
	}(addrHttp)

}

func (h *Http2Mqtt) setRestPathPrefix(prefix string) {

	ps := len(prefix)

	//remove last /
	if ps > 0 && prefix[ps-1] == '/' {
		prefix = prefix[:ps-1]
	}

	h.prefixRestApi = prefix
}

func (h *Http2Mqtt) EnableProfiling(enable bool) {

	h.profileEnable = enable
}

func (m *Http2Mqtt) SubscribeTopic(msg SubScribeMessage) error {

	t := SubScribeMessage{Qos: msg.Qos, Topic: msg.Topic}

	m.subsMutex.Lock()
	m.subs = append(m.subs, t)
	m.subsMutex.Unlock()

	return m.subscribeMessagesToBroker()

}

func (m *Http2Mqtt) subscribeMessagesToBroker() error {

	m.subsMutex.Lock()
	defer m.subsMutex.Unlock()

	for _, msg := range m.subs {
		// Go For MQTT Publish
		client := m.mqttClient
		log.Printf("Sub topic %s, Qos: %d\r\n", msg.Topic, msg.Qos)
		if token := client.Subscribe(msg.Topic, msg.Qos, m.onBrokerData); token.Error() != nil {
			// Return Error
			return token.Error()
		}
	}
	return nil
}

func (m *Http2Mqtt) onBrokerData(client MQTT.Client, msg MQTT.Message) {

	//log.Printf("* [%s] %s\n", msg.Topic(), string(msg.Payload()))

	if m.streamChannel != nil {
		m.streamChannel <- msg
	}

	//m.mqttMsgChan <- msg
	m.clients.FuncIterationForClients(func(cl *clients.Client, isConnected bool) bool {

		if isConnected {
			//casting to *chan of MQTT.Message
			ch := cl.Data["chan"].(*chan MQTT.Message)
			*ch <- msg
			return true
		}

		return false
	})

}

func (m *Http2Mqtt) onBrokerConnect(client MQTT.Client) {
	log.Println("BROKER connected!")

	m.subscribeMessagesToBroker()
}

func (m *Http2Mqtt) onBrokerDisconnect(client MQTT.Client, err error) {
	log.Println("BROKER disconnecred !", err)
}

func (m *Http2Mqtt) brokerStartConnect() {

	//on first connection library doesn't retry...do it manually
	go func(m *Http2Mqtt) {
		m.mqttClient.Connect()
		retry := time.NewTicker(30 * time.Second)
		defer retry.Stop()

		for {
			select {
			case <-retry.C:
				if !m.mqttClient.IsConnected() {
					if token := m.mqttClient.Connect(); token.Wait() && token.Error() != nil {
						log.Println("failed connection to broker retrying...")
					} else {
						return
					}
				} else {
					return
				}
			}
		}
	}(m)
}

func (m *Http2Mqtt) setupMQTT() {

	if m.mqttOpts.ClientID == "" {
		m.mqttOpts.SetClientID(getRandomClientId())
	}

	m.mqttOpts.SetAutoReconnect(true)

	//m.mqttOpts.SetConnectRetry(true)
	m.mqttOpts.SetMaxReconnectInterval(45 * time.Second)
	m.mqttOpts.SetConnectionLostHandler(m.onBrokerDisconnect)
	m.mqttOpts.SetOnConnectHandler(m.onBrokerConnect)

	client := MQTT.NewClient(m.mqttOpts)
	m.mqttClient = client
	m.brokerStartConnect()

}

func (m *Http2Mqtt) setupGin() {

	m.clients = clients.New()
	if m.Group == nil {
		m.Group = &m.Router.RouterGroup
	}
	// Ping Test
	m.Group.GET(m.prefixRestApi+"/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	//authorized := &m.Router.RouterGroup

	if m.user != "" && m.password != "" {
		m.Group = m.Router.Group("/", gin.BasicAuth(gin.Accounts{
			m.user: m.password, // user:foo password:bar
		}))
	}

	// Post To Topic
	m.Group.POST(m.prefixRestApi+"/publish", func(c *gin.Context) {
		var msg = PublishMessage{Retained: false, Qos: 0, Message: nil, Topic: ""}
		// Validate Payloadd
		if err := c.ShouldBindJSON(&msg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": err.Error()})
			return
		}
		// Go For MQTT Publish
		payload, _ := json.Marshal(msg.Message)
		client := m.mqttClient
		if token := client.Publish(msg.Topic, msg.Qos, msg.Retained, payload); token.Error() != nil {
			// Return Error
			log.Println("Error:", token.Error())
			c.JSON(http.StatusFailedDependency, gin.H{"status": "error", "error": token.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	m.Group.GET(m.prefixRestApi+"/broker", func(c *gin.Context) {

		m.subsMutex.Lock()
		defer m.subsMutex.Unlock()

		c.JSON(http.StatusOK, gin.H{"connected": m.mqttClient.IsConnected(),
			"broker": m.mqttOpts.Servers[0].Host, "user": m.mqttOpts.Servers[0].User, "subscriptions": m.subs})
	})

	m.Group.POST(m.prefixRestApi+"/subscribe", func(c *gin.Context) {
		var msg = SubScribeMessage{Qos: 0, Topic: ""}

		if !m.mqttClient.IsConnected() {
			c.JSON(http.StatusFailedDependency, gin.H{"status": "error", "error": "broker disconnected"})
			return
		}

		// Validate Payloadd
		if err := c.ShouldBindJSON(&msg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": err.Error()})
			return
		}

		m.subsMutex.Lock()
		m.subs = append(m.subs, msg)
		m.subsMutex.Unlock()

		err := m.subscribeMessagesToBroker()
		if err == nil {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		} else {
			c.JSON(http.StatusOK, gin.H{"status": "error", "error": err.Error()})
		}
		return
	})

	// Streams SSE
	m.Group.GET(m.prefixRestApi+"/ws", func(c *gin.Context) {

		data := make(clients.ClientData)

		ch := make(chan MQTT.Message, channelClientSize)
		data["chan"] = &ch //keep only the pointer here...and closinf from this function later(2 avoid copy)

		cl := m.clients.RegisterNewClientWithData(clients.WsClient, data)
		log.Println("new connection id:" + cl.Id)

		var wsUpgrader = websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}

		ws, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Errorf("Failed to set websocket upgrade: %v\n", err)
			close(ch)
			return
		}

		wsHandler := &WsHandler{id: cl.Id, ws: ws, send: &ch}

		onClose := func() {
			log.Infof("Closing ws connection")
			close(ch)
			ws.Close()
			m.clients.RemoveClient(cl)
			log.Infof("removing ws client: " + cl.Id)
		}

		go wsHandler.wsReadProcess(onClose)
		go wsHandler.wsWriteProcess(onClose)
	})

	m.Group.GET(m.prefixRestApi+"/streams", func(c *gin.Context) {

		data := make(clients.ClientData)

		ch := make(chan MQTT.Message, channelClientSize)
		data["chan"] = &ch //keep only the pointer here...and closinf from this function later(2 avoid copy)

		cl := m.clients.RegisterNewClientWithData(clients.SseClient, data)
		log.Println("new connection id:" + cl.Id)

		defer close(ch)

		for {
			c.Stream(func(w io.Writer) bool {

				select {

				case msg := <-ch:
					if msg != nil {
						if m.streamEnabled {
							c.SSEvent(msg.Topic(), msg.Payload())
						}
					} else {
						return false
					}

				case <-c.Writer.CloseNotify():
					m.clients.RemoveClient(cl)
					//close(ch)
					log.Println("removing sse client: " + cl.Id)
					return false
				}
				return true
			})
		}
	})

	if m.profileEnable {
		pprof.Register(m.Router, m.prefixRestApi+"/debug/pprof")
	}
}

func (h *WsHandler) wsWriteProcess(onClose func()) {
	defer onClose()
	for {
		select {
		case msg := <-*h.send:
			if msg != nil {
				err := h.ws.WriteMessage(websocket.TextMessage, msg.Payload())
				if err != nil {
					log.Errorf("error: %v", err)
					return
				}
			}
		}
	}
}

func (h *WsHandler) wsReadProcess(onClose func()) {
	defer onClose()
	for {
		_, message, err := h.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err) || websocket.IsCloseError(err) || err == io.EOF {
				log.Infof("Connection close client id: %s", h.id)
			} else {
				log.Errorf("error: %v", err)
			}
			break
		} else {
			log.Tracef("ws message received: %s", string(message))
		}
	}
}
