package housekeeper

import (
	"encoding/json"
	"net/http"
	"errors"
	"github.com/gorilla/websocket"
)

type s_websocketResponse struct {
	Topic string `json:"topic"`
	Message string `json:"message"`
}

type Client struct {
	hub *S_Hub
	conn *websocket.Conn
}

func (c *Client) readPump() {
	clientResponse := s_websocketResponse{}

	for {
		_, message, err := c.conn.ReadMessage()

		if err != nil {
			SharedInformation.Logger.Error(err)
			
			break
		}

		err = json.Unmarshal(message, &clientResponse)

		if err != nil {
			SharedInformation.Logger.Error(err)
			
			break
		}

		PublishMQTTMessage(clientResponse.Topic, clientResponse.Message)
	}
}

type S_Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan *S_MQTTResponse

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func newHub() *S_Hub {
	return &S_Hub{
		broadcast:  make(chan *S_MQTTResponse),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *S_Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case message := <-h.broadcast:
			packedResponse, err := json.Marshal(message)

			if err != nil {
				SharedInformation.Logger.Error(err)
				break
			}

			for client := range h.clients {
/*				SharedInformation.Logger.Error(packedResponse)
				SharedInformation.Logger.Error(client)*/
				client.conn.WriteMessage(websocket.TextMessage, packedResponse)
			}
		}
	}
}

var upgrader = websocket.Upgrader {
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func validateConfiguration() error {
	if SharedInformation.Configuration.Webserver.Protocol == "" {
		return errors.New("Webserver.Protocol missing from configuration")
	}

	if SharedInformation.Configuration.Webserver.Listen == "" {
		return errors.New("Webserver.Listen missing from configuration")
	}

/*	if configuration.Webserver.Certificate == "" {
		return errors.New("Webserver.Certificate missing from configuration")
	}

	if configuration.Webserver.CertificateKey == "" {
		return errors.New("Webserver.CertificateKey missing from configuration")
	}*/

	return nil
}

func listenForWebsocketIncoming() {
	clientResponse := s_websocketResponse{}

	for {
		for client := range SharedInformation.Hub.clients {
			_, p, err := client.conn.ReadMessage()

			if err != nil {
				SharedInformation.Logger.Error(err)
				break
			}

			err = json.Unmarshal(p, &clientResponse)

			if err != nil {
				SharedInformation.Logger.Error(err)
				break
			}

//			SharedInformation.Logger.Info(clientResponse)
//			SharedInformation.Logger.Info(clientResponse.Topic, clientResponse.Message)

			PublishMQTTMessage(clientResponse.Topic, clientResponse.Message)
		}
	}
}

func StartWebserver() error {
	err := validateConfiguration()

	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir(SharedInformation.Configuration.Webserver.WebPath))
	SharedInformation.Hub = newHub()

	go SharedInformation.Hub.run()

	mux.Handle("/", fs)
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			SharedInformation.Logger.Error(err)
			return
		}

		SharedInformation.Logger.Infof("Client connected from %s", r.RemoteAddr)

		client := &Client{hub: SharedInformation.Hub, conn: conn}

		go client.readPump()

		SharedInformation.Hub.register <- client
	})

/*	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP521,
			tls.CurveP384,
			tls.CurveP256,
		},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}*/

	srv := &http.Server{
		Addr: SharedInformation.Configuration.Webserver.Listen,
		Handler: mux,
/*		TLSConfig: cfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),*/
	}

	if err != nil {
		return err
	}

	SharedInformation.Logger.Info("Serving")

	err = srv.ListenAndServe()
//	err = srv.ListenAndServeTLS(housekeeper.Configuration.Webserver.Certificate, configuration.Webserver.CertificateKey)

	return err
}
