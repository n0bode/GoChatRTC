# Chat peer to peer (p2p) powered

# [Project in building]

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

# Test First
  ` >> go test . -v `
  - Check if all is alright, no fail
  
# How to Compile it
- If all is alright, you just run use this command in your shell intepreter
` >> chmod 777 ./run.sh `
` >> run.sh [--port] [--ip] [--db.user] [--db.password] `

# How to Initialize Database (RethinkDB)
` >> docker run --name db -p 28015:28015 -p 9000:8080 rethinkdb `

- When you start a chat-server first time, it'll set up by itself

