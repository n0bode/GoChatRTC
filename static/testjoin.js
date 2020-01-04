function createUser(){
  var s = new XMLHttpRequest();
  s.open("post", "http://" + window.location.host + "/users");
  s.send('{"name":"hsd"}');  
  s.onload = function(){
    console.log(s.responseText);
  }
}

function login(secret){
  var s = new XMLHttpRequest();
  s.open("POST", "http://" + window.location.host + "/auth/login");
  s.send(JSON.stringify({"key":secret}));
  s.onload = function(e){
    console.log(s.responseText);
  }
}

function createroom(name){
  var s = new XMLHttpRequest();
  s.open("POST", "http://" + window.location.host + "/chat/rooms");
  s.send(JSON.stringify({"name":"test"}))
  s.onload = function(e){
    console.log(s.responseText);
  };
  return s.responseText;
}

function joinroom(roomid){
  var ws = new WebSocket("ws:" + window.location.host + "/chat/rooms/" + roomid);
  ws.onopen = function(){
    var rtc = new RTCPeerConnection({
      'servers': [
        { 'urls': "stun:stun.l.google.com:19302"},
      ]
    });

    ws.onmessage = function(msg){
      console.log(msg.data);
      var offer = JSON.parse(msg.data);
      rtc.setRemoteDescription(offer);

      rtc.createAnswer().then(function(answer){
        rtc.setLocalDescription(answer);
        ws.send(JSON.stringify(answer));

        rtc.ondatachannel = function(channel){
          channel.channel.onopen = function(){
            console.log("OK");
          }
        };
      })
    };
  }
}