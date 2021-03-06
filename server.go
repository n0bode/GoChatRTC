package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"nhooyr.io/websocket/wsjson"

	"nhooyr.io/websocket"

	"./chat"
	"github.com/gorilla/mux"
	re "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

func concatStr(strs ...string) string {
	var buffer bytes.Buffer
	for _, v := range strs {
		buffer.WriteString(v)
	}
	return buffer.String()
}

func loadStaticFile(filepath string) (data []byte, err error) {
	if _, err = os.Stat(filepath); os.IsNotExist(err) {
		return
	}

	if data, err = ioutil.ReadFile(filepath); err != nil {
		return
	}
	return
}

func loadAllStaticFiles(path string) (files map[string][]byte) {
	files = make(map[string][]byte)
	filepath.Walk(path, func(filepath string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files[filepath[len(path):]], _ = loadStaticFile(filepath)
		}
		return nil
	})
	return
}

func showListRoom(session *re.Session, w http.ResponseWriter, r *http.Request) {
	cursor, err := re.DB("chat").Table("rooms").Run(session)
	if err != nil {
		log.Println(err)
		return
	}

	var room chat.Room
	rooms := make([]chat.Room, 0)
	for cursor.Next(&room) {
		rooms = append(rooms, room)
	}
	json.NewEncoder(w).Encode(rooms)
}

func setHeaderJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Accept", "application/json")
}

func configDB(session *re.Session) {
	if err := re.DBCreate("chat").Exec(session); err != nil {
		return
	}
	re.DB("chat").TableCreate("users").Exec(session)
	re.DB("chat").TableCreate("rooms").Exec(session)
}

func revalidateSession(session *re.Session, sessionKey string, user *chat.User) bool {
	cursor, err := re.DB("chat").Table("users").Filter(re.Row.Field("session").Eq(sessionKey)).Run(session)
	if err != nil {
		return false
	}

	if err = cursor.One(user); err != nil {
		return false
	}

	lastTime, _ := time.Parse(user.LastTime, time.RFC3339)
	if time.Now().Sub(lastTime).Minutes() > 5 {
		return false
	}

	re.DB("chat").Table("users").Update(map[string]interface{}{
		"lasttime": time.Now().Format(time.RFC3339),
	}).Exec(session)

	return true
}

func main() {
	//Flags to command line
	host := flag.String("host", "", "Public Address")
	port := flag.String("port", "8080", "Public Port Address")
	dbUser := flag.String("db.username", "", "Admin User RethinkDB")
	dbPassword := flag.String("db.password", "", "Admin Passwword RethinkDB")
	dbAddress := flag.String("db.address", "localhost:28015", "Address Rethinkdb")
	flag.Parse()

	//Getting Path
	path, _ := os.Executable()
	path = filepath.Join(filepath.Dir(path), "/static")
	log.Printf("Local Storage: %s", path)

	//Read Static Files
	log.Println("Reading all static file...")
	staticFiles := loadAllStaticFiles(path)
	log.Printf("Read '%d' files\n", len(staticFiles))

	//Session
	sm := chat.NewSessionManager()

	//Hub connecteds
	hub := chat.NewHub()

	//Rooms
	rooms := chat.NewRoomManager()

	//Create WebRTC Settings
	configRTC, err := json.Marshal(map[string]interface{}{
		"iceServers": []map[string]interface{}{
			map[string]interface{}{
				"urls":       []string{"stun:numb.viagenie.ca"},
				"username":   "paulorcamacan@gmail.com",
				"credential": "swordfish",
			},
			map[string]interface{}{
				"urls":       []string{"turn:numb.viagenie.ca"},
				"username":   "paulorcamacan@gmail.com",
				"credential": "swordfish",
			},
		},
	})

	//Settings DB
	session, err := re.Connect(re.ConnectOpts{
		Password: *dbPassword,
		Username: *dbUser,
		Address:  *dbAddress,
		Database: "chat",
	})

	if err != nil {
		log.Fatal(err)
	}
	configDB(session)

	//Setting Routers
	//Index route
	var route *mux.Router = mux.NewRouter()

	route.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(staticFiles["/index.html"])
	}).Methods("GET")

	route.HandleFunc("/chat.wasm", func(w http.ResponseWriter, r *http.Request) {
		w.Write(staticFiles["/chat.wasm"])
	}).Methods("GET")

	route.HandleFunc("/wasm_exec.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Application", "application/javascript")
		w.Header().Set("Accept", "application/javascript")
		w.Write(staticFiles["/wasm_exec.js"])
	}).Methods("GET")

	route.HandleFunc("/testjoin.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Application", "application/javascript")
		w.Header().Set("Accept", "application/javascript")
		w.Write(staticFiles["/testjoin.js"])
	}).Methods("GET")

	route.HandleFunc("/configRTC", func(w http.ResponseWriter, r *http.Request) {
		w.Write(configRTC)
	}).Methods("GET")

	/*** Create Room  ***/
	route.HandleFunc("/rooms/{roomID}/join", sm.Middleware(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		defer conn.Close(websocket.StatusInternalError, "Connection closed")

		ctx, close := context.WithCancel(context.Background())
		defer close()

		userID := r.Header.Get("userID")
		parms := mux.Vars(r)

		info, err := re.DB("chat").Table("rooms").Get(parms["roomID"]).Update(
			map[string]interface{}{
				"peers": re.Row.Field("peers").SetInsert(userID),
			},
		).RunWrite(session)
		if err != nil || info.Skipped == 1 {
			conn.Close(websocket.StatusInternalError, "Room not exists")
			return
		}

		room, exists := rooms.Load(parms["roomID"])
		//If the room not exists yet
		if !exists {
			room = rooms.Store(parms["roomID"])
		}

		wait := make(chan int, 1)
		fmt.Println(room.Length())
		//If already exists connected peers
		if room.Length() > 0 {
			//Set userID
			resp := map[string]interface{}{
				"event":     "create_offer",
				"requester": userID,
				"config":    string(configRTC),
			}

			//Send Offer all peer
			room.Range(func(peerID string, peer *websocket.Conn) {
				if userID == peerID {
					return
				}
				log.Println("send")
				resp["peerID"] = peerID
				if err = wsjson.Write(ctx, conn, resp); err != nil {
					room.Delete(peerID)
					re.DB("chat").Table("rooms").Get(parms["roomID"]).Update(func(d re.Term) interface{} {
						return map[string]interface{}{
							"peers": re.Row.Field("peers").Filter(func(a re.Term) interface{} {
								return a.Ne(peerID)
							}),
						}
					}).Exec(session)
					fmt.Println("erro")
					return
				}
			})
		}
		room.Store(userID, conn)

		go func() {
			ctx, cancel := context.WithCancel(context.Background())

			defer cancel()
			for {
				resp := make(map[string]interface{})
				if err := wsjson.Read(ctx, conn, &resp); err != nil {
					room.Delete(userID)
					//Remove User from Room peers list
					re.DB("chat").Table("rooms").Get(parms["roomID"]).Update(func(d re.Term) interface{} {
						return map[string]interface{}{
							"peers": d.Field("peers").Filter(func(a re.Term) interface{} {
								return a.Ne(userID)
							}),
						}
					}).RunWrite(session)
					wait <- 0
					return
				}

				switch resp["event"] {
				case "answer":
					peer, exists := room.Load(resp["requester"].(string))
					if !exists {
						continue
					}
					if err := wsjson.Write(ctx, peer, resp); err != nil {
						fmt.Println(err)
					}
					fmt.Println("Send answer")
				case "offer":
					peer, exists := room.Load(resp["peerID"].(string))
					if !exists {
						continue
					}
					if err := wsjson.Write(ctx, peer, resp); err != nil {
						fmt.Println(err)
					}
					fmt.Println("Send offer")
				}
			}
		}()
		room.Store(userID, conn)
		<-wait
	}))

	//Create room
	route.HandleFunc("/rooms", sm.Middleware(func(w http.ResponseWriter, r *http.Request) {
		setHeaderJSON(w)
		defer r.Body.Close()
		var userID string = r.Header.Get("userID")

		data := make(map[string]interface{})
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		result, err := re.DB("chat").Table("rooms").Insert(map[string]interface{}{
			"name":  data["roomName"],
			"peers": []string{},
			"owner": userID,
		}).RunWrite(session)

		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"roomID": result.GeneratedKeys[0],
		})
	})).Methods("POST")

	/* Room Get Info */
	route.HandleFunc("/rooms/{roomID}", func(w http.ResponseWriter, r *http.Request) {
		setHeaderJSON(w)
		defer r.Body.Close()

		parms := mux.Vars(r)

		cursor, err := re.DB("chat").Table("rooms").Get(parms["roomID"]).Run(session)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if cursor.IsNil() {
			log.Println(err)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		var roomInfo chat.RoomInfo
		if err = cursor.One(&roomInfo); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(roomInfo)
	}).Methods("GET")

	route.HandleFunc("/invite", sm.Middleware(func(w http.ResponseWriter, r *http.Request) {
		setHeaderJSON(w)
		var userID string = r.Header.Get("userID")

		resp := make(map[string]interface{})
		if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if userID == resp["to"] {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		inviteInfo := map[string]interface{}{
			"roomID": resp["roomID"],
			"from":   userID,
		}

		err := re.DB("chat").Table("users").Filter(re.Row.Field("userID").Eq(resp["to"])).Update(
			map[string]interface{}{
				"invites": re.Row.Field("invites").Append(inviteInfo),
			}).Exec(session)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println(err)
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		hub.WriteTo(ctx, "invite", inviteInfo, resp["to"].(string))
		w.WriteHeader(http.StatusCreated)
	})).Methods("POST")

	/*** JOIN HUB ***/
	route.HandleFunc("/hub/join", sm.Middleware(func(w http.ResponseWriter, r *http.Request) {
		var userID string = r.Header.Get("userID")

		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		defer conn.Close(websocket.StatusInternalError, "Connection closed")

		ctx, close := context.WithCancel(context.Background())
		defer close()

		data := make(map[string]interface{})
		if err = wsjson.Read(ctx, conn, &data); err != nil {
			log.Println(err)
			conn.Close(websocket.StatusInternalError, "Websocket read error")
			return
		}

		//Connected to HUB
		hub.Add(userID, conn)
		for {
			if err = wsjson.Read(ctx, conn, &data); err != nil {
				hub.Rem(userID)
				log.Printf("Disconnected from hub: %s\n", userID)
				return
			}
		}
	}))

	route.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		setHeaderJSON(w)
		defer r.Body.Close()

		info := make(map[string]interface{})
		if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println(err)
			return
		}

		cursor, err := re.DB("chat").Table("users").Filter(re.Row.Field("secretKey").Eq(info["user"])).Run(session)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if cursor.IsNil() {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var user chat.User
		if err = cursor.One(&user); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		token := sm.Store(time.Minute*15, user.UserID)
		re.DB("chat").Table("users").Get(user.ID).Update(map[string]interface{}{
			"lasttime": time.Now().Format(time.RFC3339),
		}).Exec(session)

		user.SecretKey = ""
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"session": token,
			"user":    user,
		})

	}).Methods("POST")

	route.HandleFunc("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		setHeaderJSON(w)
		defer r.Body.Close()

		var userInfo chat.User
		if err := json.NewDecoder(r.Body).Decode(&userInfo); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err := re.DB("chat").Table("session").Filter(re.Row.Field("user").Eq(userInfo.ID)).Delete().Exec(session)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Println(err)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}).Methods("POST")

	//Create a chain publicKeys
	lastPublicKey := make(chan string, 1)
	lastPublicKey <- chat.EncodeToSha(concatStr(time.Now().Add(time.Millisecond * time.Duration(rand.Int())).Format(time.RFC3339)))

	//Create User
	route.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		setHeaderJSON(w)

		var userData chat.User
		if err := json.NewDecoder(r.Body).Decode(&userData); err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if len(userData.Name) < 4 || len(userData.Name) > 10 {
			log.Println("Name not valid")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		matched, _ := regexp.Match("^([$][A-Za-z]|([a-zA-Z]))([_.]?[a-zA-Z0-9])*$", []byte(userData.Name))
		if !matched {
			log.Println("Name not valid")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		userData.Created = time.Now().Format(time.RFC3339)
		res, err := re.DB("chat").Table("users").Insert(map[string]interface{}{
			"name":    userData.Name,
			"created": userData.Created,
			"invites": []string{},
			"friends": []string{},
		}).RunWrite(session)

		if err != nil {
			log.Println(err)
		}

		userData.UserID = chat.EncodeToSha(concatStr(res.GeneratedKeys[0], userData.Name, userData.Created))
		userData.SecretKey = chat.EncodeToSha(concatStr(userData.UserID, <-lastPublicKey))
		lastPublicKey <- userData.UserID

		re.DB("chat").Table("users").Get(res.GeneratedKeys[0]).Update(map[string]interface{}{
			"userID":    userData.UserID,
			"secretKey": userData.SecretKey,
		}).Exec(session)

		userData.ID = ""
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(&userData); err != nil {
			log.Println(err)
		}
	}).Methods("POST")

	route.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		cursor, err := re.DB("chat").Table("users").Filter(re.Row.Field("id").Fill()).Run(session)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var users []chat.User
		var user chat.User
		for cursor.Next(&user) {
			users = append(users, user)
		}
		json.NewEncoder(w).Encode(users)
	}).Methods("GET")

	route.HandleFunc("/users/{userID}/addfriend", sm.Middleware(func(w http.ResponseWriter, r *http.Request) {
		var userID string = r.Header.Get("userID")
		defer r.Body.Close()

		info := make(map[string]interface{})
		if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		writeInfo, err := re.DB("chat").Table("users").Filter(re.Row.Field("userID").Eq(userID)).Update(map[string]interface{}{
			"friends": re.Row.Field("friends").SetInsert(info["user"]),
		}).RunWrite(session)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if writeInfo.Unchanged == 1 {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})).Methods("POST")

	route.HandleFunc("/users/{userID}", func(w http.ResponseWriter, r *http.Request) {
		parms := mux.Vars(r)

		if userID, ok := parms["userID"]; ok {
			cursor, err := re.DB("chat").Table("users").Filter(re.Row.Field("userID").Eq(userID)).Run(session)
			if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			var user chat.User
			if err = cursor.One(&user); err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			//Hide some fields
			user.SecretKey = ""
			user.Invites = nil

			if err = json.NewEncoder(w).Encode(&user); err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}).Methods("GET")

	//Handler for IMAGES
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Path
		if len(url) >= 4 {
			switch url[len(url)-4:] {
			case ".png", ".jpg", ".gif", ".svg", ".ico":
				if img, ok := staticFiles[concatStr("/imgs", url)]; ok {
					w.Write(img)
					return
				}
				return
			}
		}
		route.ServeHTTP(w, r)
	})

	//Update static files
	go func() {
		mods := make(map[string]time.Time)
		for {
			time.Sleep(time.Second)
			filepath.Walk(path, func(filepath string, info os.FileInfo, err error) error {
				var addr string = filepath[len(path):]
				if val, exists := mods[addr]; exists {
					if val != info.ModTime() {
						log.Printf("Reload File '%s'\n", addr)
						staticFiles[addr], _ = loadStaticFile(filepath)
					}
				}
				mods[addr] = info.ModTime()
				return nil
			})
		}
	}()

	//Listening Server
	address := concatStr(*host, ":", *port)
	log.Printf("Server started on %s\n", address)
	if err := http.ListenAndServe(address, handler); err != nil {
		log.Fatal(err)
	}
}
