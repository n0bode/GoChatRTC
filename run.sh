HOST=
PORT=8080
DB_USER=
DB_PASSWORD=
DB_ADDRESS=localhost:28015

echo "Building Server..."
if ! go build -o sv server.go; then
  exit 1
fi

echo "Building WASM chat..."
if ! GOOS=js GOARCH=wasm go build -o static/chat.wasm webassembly/chatwasm.go; then
  exit 1
fi
echo "Executing Server"
./sv -host "$HOST" -port "$PORT"
rm ./sv
rm static/chat.wasm