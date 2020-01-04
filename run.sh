echo "Building Server..."
if ! go build -o sv server.go; then
  exit 1
fi

echo "Building WASM chat..."
if ! GOOS=js GOARCH=wasm go build -o static/chat.wasm chatwasm.go; then
  exit 1
fi
echo "Executing Server"
./sv
rm ./sv