package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"syscall/js"
	"time"

	"github.com/pion/webrtc"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

var (
	CONFIG webrtc.Configuration
	HOST   string
)

func init() {
	CONFIG = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			webrtc.ICEServer{
				URLs: []string{
					"stun:stun.l.google.com:19302",
				},
			},
		},
	}
	HOST = js.Global().Get("document").Get("location").Get("host").String()
}

func concatStr(args ...string) (str string) {
	var buffer bytes.Buffer
	defer func() {
		str = buffer.String()
	}()

	for _, val := range args {
		buffer.WriteString(val)
	}
	return
}

func concatURL(prefix string, suffixes ...string) string {
	buffer := bytes.NewBufferString(prefix)
	buffer.WriteString("://")
	buffer.WriteString(HOST)
	for _, suf := range suffixes {
		if suf[0] != '/' {
			buffer.WriteString("/")
		}
		if suf[len(suf)-1] == '/' {
			suf = suf[:len(suf)-1]
		}
		buffer.WriteString(suf)
	}
	return buffer.String()
}

func join(roomID string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, concatURL("ws", "/chat/rooms/", roomID), nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	peer, err := webrtc.NewPeerConnection(CONFIG)
	if err != nil {
		fmt.Println(err)
		return
	}

	var offer webrtc.SessionDescription
	if err := wsjson.Read(ctx, conn, &offer); err != nil {
		fmt.Println(err)
		return
	}

	if err = peer.SetRemoteDescription(offer); err != nil {
		fmt.Println(err)
		return
	}

	answer, err := peer.CreateAnswer(nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	if err = peer.SetLocalDescription(answer); err != nil {
		fmt.Println(err)
		return
	}

	wsjson.Write(ctx, conn, answer)
	peer.OnDataChannel(func(channel *webrtc.DataChannel) {
		channel.OnOpen(func() {
			fmt.Println("Connected")
		})
	})
}

func post(url string, data map[string]interface{}, callback func(status int, response string)) {
	req := js.Global().Get("XMLHttpRequest").New(nil)

	req.Set("onload", js.FuncOf(func(v js.Value, args []js.Value) interface{} {
		callback(req.Get("status").Int(), req.Get("response").String())
		return nil
	}))

	req.Call("open", "POST", url)
	encode, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("Post '%s'\n", url)
		return
	}
	req.Call("send", string(encode))
}

func createUser(username string) {
	post(concatURL("http", "/users"), map[string]interface{}{
		"name": username,
	}, func(status int, data string) {
		if status == http.StatusCreated {
			fmt.Println(data)
		}
	})
}

func setCookie(values ...string) {
	buffer := &bytes.Buffer{}
	for i := 0; i < len(values); i += 2 {
		buffer.WriteString(values[i+0])
		buffer.WriteString("=")
		buffer.WriteString(values[i+1])
		buffer.WriteString(";")
	}
	fmt.Println(buffer.String())
	js.Global().Get("document").Set("cookie", buffer.String())
}

func notifyInvite(from string, roomid string) {
	js.Global().Call("alert", concatStr("You recv a invite from ", from, " to join ", roomid))
}

func sendMessage(message interface{}, to string) {

}

func connectHub(session, userID string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, _, err := websocket.Dial(ctx, concatURL("ws", "/hub"), nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close(websocket.StatusInternalError, "Connection closed")

	if err := wsjson.Write(ctx, conn, map[string]interface{}{
		"session": session,
		"userID":  userID,
	}); err != nil {
		fmt.Println(err)
		return
	}

	data := make(map[string]interface{})
	for {
		if err = wsjson.Read(ctx, conn, &data); err != nil {
			fmt.Println(err)
			return
		}

		switch data["event"] {
		case "invite":
			invite := data["message"].(map[string]interface{})
			notifyInvite(invite["from"].(string), invite["roomID"].(string))
			break
		default:
			conn.Close(websocket.StatusInternalError, "Error")
		}
	}
}

func loginUser(secretKey string) {
	post(concatURL("http", "/auth/login"), map[string]interface{}{
		"secretKey": secretKey,
	}, func(status int, response string) {
		if status == http.StatusCreated {
			resp := make(map[string]interface{})
			if err := json.Unmarshal([]byte(response), &resp); err != nil {
				fmt.Println(err)
				return
			}

			session := resp["session"].(string)
			userID := resp["user"].(map[string]interface{})["userID"].(string)

			//Connect to hub
			go connectHub(session, userID)
		}
	})
}

func registerFuncs() {
	js.Global().Set("join", js.FuncOf(func(v js.Value, args []js.Value) interface{} {
		if len(args) != 1 {
			fmt.Println("Missing args to join")
			return nil
		}
		go join(args[0].String())
		return nil
	}))

	//Create user callback
	js.Global().Set("createuser", js.FuncOf(func(v js.Value, args []js.Value) interface{} {
		if len(args) != 1 {
			fmt.Println("Missing args to create a new user")
			return nil
		}
		createUser(args[0].String())
		return nil
	}))

	//Login Callback
	js.Global().Set("login", js.FuncOf(func(v js.Value, args []js.Value) interface{} {
		if len(args) != 1 {
			fmt.Println("Missing args to log-in")
			return nil
		}
		loginUser(args[0].String())
		return nil
	}))
}

func main() {
	ch := make(chan int, 0)
	registerFuncs()
	<-ch
}
