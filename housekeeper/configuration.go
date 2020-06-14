package housekeeper

import(
	"github.com/op/go-logging"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type S_configuration struct {
	MQTT struct {
		Broker string `json:"broker"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"mqtt"`
	Webserver struct {
		Protocol string `json:"protocol"`
		Listen string `json:"listen"`
		WebPath string `json:"web_path"`
		Certificate string `json:"certificate"`
		CertificateKey string `json:"certificate_key"`
//		LogRequests bool `json:"log_requests"`
	} `json:"webserver"`
}

type sharedInformation struct {
	Logger *logging.Logger
	MQTTClient MQTT.Client
	Configuration *S_configuration
}

var SharedInformation = sharedInformation{ nil, nil, nil }