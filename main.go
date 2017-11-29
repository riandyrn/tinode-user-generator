package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsEndpoint     = "ws://127.0.0.1:6060/v0/channels?apikey=AQEAAAABAAD_rAp4DJh05a1HAwFT3A6K"
	wsReadTimeout  = time.Duration(2 * time.Second)
	wsWriteTimeout = time.Duration(2 * time.Second)
)

func main() {
	nUsers := flag.Int("n", 10, "number of target generated users")
	outputPath := flag.String("output", "./tokens.csv", "path of output file")
	flag.Parse()
	if err := generateUsers(*nUsers, *outputPath); err != nil {
		log.Panicf("unable to generate users due: %v", err)
	}
}

func generateUsers(n int, outputPath string) (err error) {
	// prepare registration for each user
	timestamp := time.Now().UTC().Unix()
	var registrations []string
	for i := 0; i < n; i++ {
		username := fmt.Sprintf("usr%v_%v", timestamp, i)
		secret := generateSecret(username + ":" + username)
		registration := map[string]interface{}{
			"acc": map[string]interface{}{
				"id":     "register user",
				"user":   "new",
				"scheme": "basic",
				"secret": secret,
				"login":  true,
				"desc": map[string]interface{}{
					"public": map[string]string{"fn": username},
				},
				"tags": []string{fmt.Sprintf("%v@example.com", username)},
			},
		}
		b, err := json.Marshal(registration)
		if err != nil {
			continue
		}
		registrations = append(registrations, string(b))
	}
	if len(registrations) > 0 {
		// register each user, save generated token
		var tokens []string
		for i := 0; i < len(registrations); i++ {
			// initialize websocket
			wsClient, err := initializeWebsocket()
			if err != nil {
				log.Printf("unable to initiate ws connection due: %v", err)
				continue
			}
			token, err := sendRegistration(wsClient, registrations[i])
			if err != nil {
				log.Println(err)
				continue
			}
			log.Printf("generated token: %v", token)
			tokens = append(tokens, token)
			closeWebsocket(wsClient)
		}
		// write tokens to file
		if err = writeTokensToFile(tokens, outputPath); err != nil {
			return fmt.Errorf("unable to write tokens to file due: %v", err)
		}
	}
	return nil
}

func generateSecret(authStr string) string {
	buf := make([]byte, base64.URLEncoding.EncodedLen(len(authStr)))
	base64.URLEncoding.Encode(buf, []byte(authStr))
	return string(buf)
}

func writeTokensToFile(tokens []string, outputPath string) error {
	content := strings.Join(tokens, "\n")
	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return fmt.Errorf("unable to open file due: %v", err)
	}
	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("unable to write tokens to file due: %v", err)
	}
	return nil
}

// create websocket connection, initialize connection
func initializeWebsocket() (*websocket.Conn, error) {
	wsClient, _, err := websocket.DefaultDialer.Dial(wsEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to initiate ws client due: %v", err)
	}
	hi := map[string]interface{}{
		"hi": map[string]interface{}{
			"id":  "init connection",
			"ver": "0.13",
			"ua":  "TinodeWeb/0.13 (MacIntel) tinodejs/0.13",
		},
	}
	p, _ := json.Marshal(hi)
	_, err = sendWsTextMessage(wsClient, p)
	if err != nil {
		return nil, fmt.Errorf("unable to send {hi} message due: %v", err)
	}
	return wsClient, nil
}

func closeWebsocket(wsClient *websocket.Conn) {
	wsClient.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	)
	wsClient.Close()
}

func sendRegistration(wsClient *websocket.Conn, registration string) (token string, err error) {
	res, err := sendWsTextMessage(wsClient, []byte(registration))
	if err != nil {
		return "", fmt.Errorf("unable to send registration due: %v", err)
	}
	var resParsed RegistrationResponse
	if err = json.Unmarshal(res, &resParsed); err != nil {
		return "", fmt.Errorf("unable to parse response due: %v", err)
	}
	token = resParsed.Ctrl.Params.Token
	if len(token) > 0 {
		return token, nil
	}
	return "", fmt.Errorf("no token found on response")
}

func sendWsTextMessage(c *websocket.Conn, message []byte) (res []byte, err error) {
	// send command
	c.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
	if err = c.WriteMessage(websocket.TextMessage, message); err != nil {
		return nil, err
	}
	// parse the response
	c.SetReadDeadline(time.Now().Add(wsReadTimeout))
	_, res, err = c.ReadMessage()
	if err != nil {
		return nil, err
	}
	return res, nil
}
