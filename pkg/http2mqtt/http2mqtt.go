package http2mqtt

import (
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/freedreamer82/go-http2mqtt/pkg/sseClients"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	//"time"
)

const MaxClientIdLen = 10

func (s *Http2Mqtt) SetGinAuth(user string, password string) {
	s.user = user
	s.password = password
}

// PublishMessage is the sample data transfer object for publish http route
type PublishMessage struct {
	Topic    string `binding:"required" json:"topic"`
	Message  string `binding:"required" json:"data"`
	Qos      byte   `json:"qos"`
	Retained bool   `json:"retained"`
}

type SubScribeMessage struct {
	Topic string `binding:"required" json:"topic"`
	Qos   byte   `json:"qos"`
}

type Http2Mqtt struct {
	Router        *gin.Engine
	MqttBrokerURL string
	mqttClient    MQTT.Client
	mqttOpts      *MQTT.ClientOptions
	user          string
	password      string
	sseCLients    *sseClients.SseCLients
	subsMutex     sync.Mutex
	subs          []SubScribeMessage
	profileEnable bool
	prefixRestApi string
}

func getRandomClientId() string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, MaxClientIdLen)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return "http2mqtt-" + string(bytes)
}

func New(mqttOpts *MQTT.ClientOptions) *Http2Mqtt {

	h := Http2Mqtt{Router: nil, MqttBrokerURL: "", mqttOpts: nil, user: "", password: "", profileEnable: false, prefixRestApi: ""}

	router := gin.New()
	h.Router = router
	h.mqttOpts = mqttOpts

	h.setupMQTT()

	return &h
}

func NewWithRouter(mqttOpts *MQTT.ClientOptions, router *gin.Engine) *Http2Mqtt {
	h := Http2Mqtt{Router: nil, MqttBrokerURL: "", mqttOpts: nil, user: "", password: "", profileEnable: false, prefixRestApi: ""}

	h.Router = router
	h.mqttOpts = mqttOpts

	h.setupMQTT()

	return &h
}

func (h *Http2Mqtt) Run(addrHttp string) {

	h.setupGin()
	defer func(add string) {
		go h.Router.Run(add)
	}(addrHttp)

}

func (h *Http2Mqtt) SetRestPathPrefix(prefix string) {

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

	log.Printf("* [%s] %s\n", msg.Topic(), string(msg.Payload()))
	//m.mqttMsgChan <- msg
	l := m.sseCLients.GetClientsList()
	for ssecl, isConnected := range l {

		if isConnected {
			//casting to *chan of MQTT.Message
			ch := ssecl.Data["chan"].(*chan MQTT.Message)
			*ch <- msg
		}
	}
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
				if ! m.mqttClient.IsConnected() {
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

	m.sseCLients = sseClients.New()
	noauthorized := m.Router

	// Ping Test
	noauthorized.GET(m.prefixRestApi+"/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	authorized := &m.Router.RouterGroup

	if m.user != "" && m.password != "" {
		authorized = m.Router.Group("/", gin.BasicAuth(gin.Accounts{
			m.user: m.password, // user:foo password:bar
		}))
	}

	// Post To Topic
	authorized.POST(m.prefixRestApi+"/publish", func(c *gin.Context) {
		var msg = PublishMessage{Retained: false, Qos: 0, Message: "", Topic: ""}
		// Validate Payloadd
		if err := c.ShouldBindJSON(&msg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": err.Error()})
			return
		}
		// Go For MQTT Publish
		client := m.mqttClient
		if token := client.Publish(msg.Topic, msg.Qos, msg.Retained, msg.Message); token.Error() != nil {
			// Return Error
			log.Println("Error:", token.Error())
			c.JSON(http.StatusFailedDependency, gin.H{"status": "error", "error": token.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	authorized.GET(m.prefixRestApi+"/broker", func(c *gin.Context) {

		m.subsMutex.Lock()
		defer m.subsMutex.Unlock()

		c.JSON(http.StatusOK, gin.H{"connected": m.mqttClient.IsConnected(),
			"broker": m.mqttOpts.Servers[0].Host, "user": m.mqttOpts.Servers[0].User, "subscriptions": m.subs})
	})

	authorized.POST(m.prefixRestApi+"/subscribe", func(c *gin.Context) {
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
		if (err == nil) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		} else {
			c.JSON(http.StatusOK, gin.H{"status": "error", "error": err.Error()})
		}
		return
	})

	// Streams SSE
	authorized.GET(m.prefixRestApi+"/streams", func(c *gin.Context) {

		data := make(sseClients.SseClientData)

		ch := make(chan MQTT.Message, 5)
		data["chan"] = &ch //keep only the pointer here...and closinf from this function later(2 avoid copy)

		cl := m.sseCLients.RegisterNewCLientWithData(data)
		log.Println("new connection id:" + cl.Id)

		defer close(ch)

		for {
			c.Stream(func(w io.Writer) bool {

				select {

				case msg := <-ch:
					if msg != nil {
						c.SSEvent(msg.Topic(), msg.Payload())
					} else {
						return false
					}

				case <-c.Writer.CloseNotify():
					m.sseCLients.RemoveCLient(cl)
					//close(ch)
					log.Println("removing sse client: " + cl.Id)
					return false
				}
				return true
			})
		}
	})

	if (m.profileEnable) {
		pprof.Register(m.Router, m.prefixRestApi+"/debug/pprof")
	}
}
