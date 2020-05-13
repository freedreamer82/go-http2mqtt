package main

import (
	//	"github.com/go-delve/delve/pkg/config"

	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"http2mqtt/internal/pkg/config"
	"http2mqtt/pkg/http2mqtt"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"net/http"
	"os"
)

var (
	_app = kingpin.New("htt2mqtt", "bridge between http and mqtt")

	_logFile        = _app.Flag("log", "log path file").Short('l').String()
	_logFolder      = _app.Flag("logFolder", "log path").Default("").Short('f').String()
	_logEnabled     = _app.Flag("debug", "").Short('d').Bool()
	_configFile     = _app.Flag("config", "config TOML file ").Short('c').String()
	_restAPIhost    = _app.Flag("rest config", "rest EndPoint host IP:PORT").Short('r').Default("false").Bool()
	_host           = _app.Arg("host", "http server URL with port").Required().String()
	_broker         = _app.Arg("broker", "Broker mqtt URL ip:port").Required().String()
	_user           = _app.Flag("user", "username").Short('u').Default("").String()
	_password       = _app.Flag("password", "password").Short('p').Default("").String()
	_profileEnabled = _app.Flag("pprof debug ", "pprof on /debug/pprof/profile").Short('i').Default("false").Bool()

	//Global vars
	_appConfig *config.Config
	_log       = logrus.New()
	_router    *gin.Engine
	_bridge    *http2mqtt.Http2Mqtt
)

func startWebServer(router *gin.Engine, addr string) {
	_log.Info("Starting Web server on ", addr)
	router.Run(addr)
}

func getRestConfig(w http.ResponseWriter, req *http.Request) {
	if _appConfig != nil {
		json.NewEncoder(w).Encode(*_appConfig)
	}
}

func setRestConfig(w http.ResponseWriter, req *http.Request) {
	if _appConfig != nil {
		params := mux.Vars(req)
		fmt.Println(params)

		appConfig := config.NewCopy(_appConfig)
		//		err = json.Unmarshal(reqBody, &appConfig)
		err := json.NewDecoder(req.Body).Decode(&appConfig)
		fmt.Print(appConfig)
		if err != nil {
			_log.Error("failed parsing json...")
			return
		}

		//substitute actual config with the new one
		_appConfig = &appConfig
	}
}

func main() {

	//gin.SetMode(gin.ReleaseMode)

	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
	}
	logrus.SetFormatter(formatter)

	_app.HelpFlag.Short('h')
	switch kingpin.MustParse(_app.Parse(os.Args[1:])) {
	//no command for now
	// case "":
	}

	if *_configFile != "" {
		conf, err := config.Parse(*_configFile)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		_appConfig = conf
	} else {
		conf := config.Default()
		_appConfig = &conf
	}

	if _appConfig.LogEnabled || *_logEnabled {
		_log.Level = _appConfig.LogLevel.Level
		if *_logFile != "" {
			file, err := os.OpenFile(*_logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err == nil {
				_log.Out = file
			} else {
				_log.Info("Failed to _log to file, using default stderr")
			}
		} else {
			_log.Out = os.Stdout
		}
	} else {
		_log.Out = ioutil.Discard
	}

	_appConfig.PrintDebug(_log)

	opts := mqtt.ClientOptions{}
	opts.AddBroker("tcp://" + *_broker)

	host := *_host
	if *_restAPIhost != false {
		_router = gin.New()

		_router.GET("/config", gin.WrapF(getRestConfig))
		_router.POST("/config", gin.WrapF(setRestConfig))
		_bridge = http2mqtt.NewWithRouter(&opts, _router)

	} else {
		_bridge = http2mqtt.New(&opts)
	}

	if *_user != "" && *_password != "" {
		_bridge.SetGinAuth("user", "pass")
	}

	if (*_profileEnabled) {
		_bridge.EnableProfiling(true)
	}
	_bridge.Run(host)

	select {} //wait forever
}
