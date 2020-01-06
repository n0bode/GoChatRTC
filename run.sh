HOST=
PORT=8080
DB_USER=
DB_PASSWORD=
DB_ADDRESS=localhost:28015

echo "Building Server..."
if ! go build -o sv.bin server.go; then
  exit 1
fi

echo "Building WASM chat..."
if ! GOOS=js GOARCH=wasm go build -o static/chat.wasm webassembly/chatwasm.go; then
  exit 1
fi
echo "Executing Server"

clear(){
  rm ./sv.bin
  rm static/chat.wasm
  exit
}

while [ ! $# -eq 0 ]
do
  case $1 in
  --port|-p) PORT=$2;;
  --host|-h) HOST=$2;;
  --db.user) DB_USER=$2;;
  --db.password) DB_PASSWORD=$2;;
  --db.address) DB_ADDRESS=$2;;
  esac
  shift
done

./sv.bin -host "$HOST" -port "$PORT" -db.username "$DB_USER" -db.password "$DB_PASSWORD" -db.address "$DB_ADRESS"
clear()