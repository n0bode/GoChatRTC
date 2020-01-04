# Chat peer to peer (p2p) powered

- RethinkDB as Main Database
- Golang as Main Language
- Html/Css as Front
- WebAssembly as Client-Side/Communication

# Golang packages

` >> go get github.com/gorilla/mux `
- Mux is necessary to create RESTful

` >> go get github.com/pion/webrtc `
- Pion Webrtc is necessary to create communication between the peers with WebRTC Connection

` >> go get nhooyr.io/websocket`
- Nhoor Websocket is necessary to create communication between the server and peers
- Peers need to send Offer and Answer between them, but they need a stabilished communication to do it
- Then, WebSocket creates this communication

# How to Initialize Database (RethinkDB)
`console
  >> docker run --name db -p 28015:28015 -p 9000:8080 rethinkdb
`
- When you start a chat-server first time, it'll set up by itself

