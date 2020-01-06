package main

import (
	"bytes"
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"nhooyr.io/websocket/wsjson"

	"nhooyr.io/websocket"

	"./chat"
	"github.com/pion/webrtc"
)

/*
func TestJoinHub(t *testing.T) {
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/hub/join", nil)
	if err != nil {
		t.Fatal(err)
	}

	peer, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			webrtc.ICEServer{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		conn.Close()
		peer.Close()
	}()

	var offer webrtc.SessionDescription
	if err := conn.ReadJSON(&offer); err != nil {
		t.Fatal(err)
	}

	if err := peer.SetRemoteDescription(offer); err != nil {
		t.Fatal(err)
	}

	answer, err := peer.CreateAnswer(nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := peer.SetLocalDescription(answer); err != nil {
		t.Fatal(err)
	}
	conn.WriteJSON(answer)

	//Channel for wait datachannel connected
	wait := make(chan interface{})
	peer.OnDataChannel(func(channel *webrtc.DataChannel) {
		channel.OnMessage(func(data webrtc.DataChannelMessage) {
			t.Run("Send Message", func(t *testing.T) {
				SendMessage(channel, t)
				wait <- 0
			})
		})
	})
	<-wait
}
*/

func SendMessage(channel *webrtc.DataChannel, t *testing.T) {
	if err := channel.Send([]byte("testing")); err != nil {
		t.Fatal(err)
	}
}

func TestCreateUser(t *testing.T) {
	var nickname string = genRandNickname()
	t.Logf("Creating User: %s\n", nickname)
	createUser(nickname, t)
}

func TestSendInviteRoom(t *testing.T) {
	users := make(chan chat.User, 2)

	t.Run("CreateUser0", func(t0 *testing.T) {
		var nickname string = genRandNickname()
		users <- createUser(nickname, t0)
	})

	t.Run("CreateUser1", func(t0 *testing.T) {
		var nickname string = genRandNickname()
		users <- createUser(nickname, t0)
	})

	var from chat.User = <-users
	var to chat.User = <-users

	session := make(chan string, 1)

	t.Run("Login", func(t0 *testing.T) {
		session <- loginUser(from.SecretKey, t0)
	})

	var buffer bytes.Buffer
	invite := map[string]interface{}{
		"to":     to.UserID,
		"roomID": "",
	}

	if err := json.NewEncoder(&buffer).Encode(invite); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "http://localhost:8080/invite", &buffer)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", <-session)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Fail()
	}
}

func TestCreateRoom(t *testing.T) {
	user := make(chan chat.User, 1)
	t.Run("CreateUser", func(t0 *testing.T) {
		user <- createUser(genRandNickname(), t0)
	})

	session := make(chan string, 1)
	t.Run("Login", func(t0 *testing.T) {
		session <- loginUser((<-user).SecretKey, t0)
	})

	var roomID string = createRoom(<-session, "room101", t)

	resp, err := http.Get("http://localhost:8080/rooms/" + roomID)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fail()
	}
}

func TestJoinHub(t *testing.T) {
	user := make(chan chat.User, 1)
	t.Run("CreateUser", func(t0 *testing.T) {
		user <- createUser(genRandNickname(), t0)
	})

	session := make(chan string, 1)
	t.Run("Login", func(t0 *testing.T) {
		session <- loginUser((<-user).SecretKey, t0)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	header := make(http.Header)
	header.Set("Authorization", <-session)
	conn, _, err := websocket.Dial(ctx, "ws://localhost:8080/hub/join", &websocket.DialOptions{
		HTTPHeader: header,
	})

	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(websocket.StatusInternalError, "Disconnected is normal")
}

func TestJoinRoom(t *testing.T) {
	users := make(chan chat.User, 2)

	t.Run("CreateUser0", func(t0 *testing.T) {
		users <- createUser(genRandNickname(), t0)
	})

	t.Run("CreateUser1", func(t0 *testing.T) {
		users <- createUser(genRandNickname(), t0)
	})

	t.Run("Login0", func(t0 *testing.T) {
		t0.Parallel()
		var token string = loginUser((<-users).SecretKey, t0)
		t.Run("Join Room 0", func(t1 *testing.T) {
			joinRoom(token, "b22b1189-2332-40af-8fb2-df84e637cbdf", t1)
		})
	})

	t.Run("Login1", func(t0 *testing.T) {
		t0.Parallel()
		var token string = loginUser((<-users).SecretKey, t0)

		t.Run("Join Room 1", func(t1 *testing.T) {
			joinRoom(token, "b22b1189-2332-40af-8fb2-df84e637cbdf", t1)
		})
	})
}

func createRoom(token, roomName string, t *testing.T) string {
	var buffer bytes.Buffer
	json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"roomName": roomName,
	})

	req, err := http.NewRequest("POST", "http://localhost:8080/rooms", &buffer)
	if err != nil {
		t.Fail()
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Error(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatal("Error to create room")
	}

	data := make(map[string]interface{})
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Error(err)
	}
	return data["roomID"].(string)
}

func genRandNickname() (nickname string) {
	var r int = int('z') - int('a')
	for i := 0; i < 8; i++ {
		nickname += string(rune(int8('a') + int8(rand.Intn(r))))
	}
	return
}

func createUser(nickname string, t *testing.T) (user chat.User) {
	var buffer bytes.Buffer
	auth := map[string]interface{}{
		"name": nickname,
	}

	//Initialize
	rand.Seed(time.Now().Unix())
	if err := json.NewEncoder(&buffer).Encode(auth); err != nil {
		t.Fatal(err)
	}

	resp, err := http.Post("http://localhost:8080/users", "application/json", &buffer)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatal("Error to create user")
	}

	if err = json.NewDecoder(resp.Body).Decode(&user); err != nil {
		t.Fatal(err)
	}
	return user
}

func loginUser(secret string, t *testing.T) string {
	var buffer bytes.Buffer
	auth := map[string]interface{}{
		"user": secret,
	}

	if err := json.NewEncoder(&buffer).Encode(auth); err != nil {
		t.Error(err)
	}

	resp, err := http.Post("http://localhost:8080/auth/login", "application/json", &buffer)
	if err != nil {
		t.Error(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Error("Error to request")
	}

	data := make(map[string]interface{})
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Error(err)
	}
	return data["session"].(string)
}

func joinRoom(token, roomID string, t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	header := make(http.Header)
	header.Set("Authorization", token)

	conn, _, err := websocket.Dial(ctx, "ws://localhost:8080/rooms/"+roomID+"/join", &websocket.DialOptions{
		HTTPHeader: header,
	})

	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(websocket.StatusInternalError, "Disconnected")
	var sucess bool

	peers := make(map[string]*webrtc.PeerConnection)
	for {
		resp := make(map[string]interface{})
		if err := wsjson.Read(ctx, conn, &resp); err != nil {
			if sucess {
				return
			}
			t.Fatal(err)
		}

		switch resp["event"] {
		case "create_offer":
			peer, err := webrtc.NewPeerConnection(webrtc.Configuration{})
			if err != nil {
				t.Fatal(err)
			}

			offer, err := peer.CreateOffer(nil)
			if err != nil {
				t.Fatal(err)
			}

			if err = peer.SetLocalDescription(offer); err != nil {
				t.Fatal(err)
			}

			resp["event"] = "offer"
			resp["offer"] = offer
			if err = wsjson.Write(ctx, conn, resp); err != nil {
				t.Fatal(err)
			}
			peers[resp["peerID"].(string)] = peer
			//fmt.Println("Created offer")
		case "offer":
			//fmt.Println("Recv offer")
			peer, err := webrtc.NewPeerConnection(webrtc.Configuration{})
			if err != nil {
				t.Fail()
			}

			data, err := json.Marshal(resp["offer"])
			if err != nil {
				t.Fatal(err)
			}
			delete(resp, "offer")

			var offer webrtc.SessionDescription
			if err = json.Unmarshal(data, &offer); err != nil {
				t.Fatal(err)
			}

			if err = peer.SetRemoteDescription(offer); err != nil {
				t.Fatal(err)
			}

			answer, err := peer.CreateAnswer(nil)
			if err != nil {
				t.Fatal(err)
			}

			if err = peer.SetLocalDescription(answer); err != nil {
				t.Fatal(err)
			}

			peer.OnDataChannel(func(room *webrtc.DataChannel) {
				room.OnOpen(func() {
					room.SendText("Sucess")
				})

				room.OnMessage(func(msg webrtc.DataChannelMessage) {
					sucess = true
					conn.Close(websocket.StatusGoingAway, "Sucess")
				})
			})

			resp["event"] = "answer"
			resp["answer"] = answer
			if err = wsjson.Write(ctx, conn, resp); err != nil {
				t.Fatal(err)
			}
		case "answer":
			peer, ok := peers[resp["peerID"].(string)]
			if !ok {
				t.Fatal("Missing peer")
			}

			data, err := json.Marshal(resp["answer"])
			if err != nil {
				t.Fatal(err)
			}
			delete(resp, "offer")

			var answer webrtc.SessionDescription
			if err = json.Unmarshal(data, &answer); err != nil {
				t.Fatal(err)
			}

			if err := peer.SetRemoteDescription(answer); err != nil {
				t.Fatal(err)
			}

			room, err := peer.CreateDataChannel("rrom", nil)
			if err != nil {
				t.Fatal(err)
			}

			room.OnOpen(func() {
				room.SendText("Sucess")
			})

			room.OnMessage(func(msg webrtc.DataChannelMessage) {
				sucess = true
				conn.Close(websocket.StatusGoingAway, "Sucess")

			})
		}
	}
}
