package natsclient

import (
	"log"

	"github.com/nats-io/nats.go"
)

// Username and Password
func AuthUsernamePasswordConnect(username, password, server string) (*nats.Conn, error) {
	log.Printf("Connecting to NATS server with username and password ...")

	nc, err := nats.Connect(server, nats.UserInfo(username, password))

	// Option2
	// serverUrl := fmt.Sprintf("nats://%s:%s@%s", username, password, server)

	// nc, err := nats.Connect(serverUrl)

	if err != nil {
		return nil, err
	}

	log.Printf("Connected to server: %s", nc.ConnectedUrl())
	log.Printf("Discovered servers: %v", nc.DiscoveredServers())

	return nc, nil
}

// Token
func AuthTokenConnect(token, server string) (*nats.Conn, error) {
	log.Printf("Connecting to NATS server with token ...")

	nc, err := nats.Connect(server, nats.Token(token))

	if err != nil {
		return nil, err
	}

	log.Printf("Connected to server: %s", nc.ConnectedUrl())
	log.Printf("Discovered servers: %v", nc.DiscoveredServers())

	return nc, nil
}
