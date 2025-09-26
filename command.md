./bin/gateway --config configs/gateway.yaml  
consul agent -dev -ui -bind 127.0.0.1
cd ~/Documents/Fun/isA_user && source .venv/bin/activate && python -m microservices.account_service.main


cd ~/Documents/Fun/isA_Cloud && go build -o bin/gateway ./cmd/gateway && ./bin/gateway --config configs/gateway.yaml

cd ~/Documents/Fun/isA_Cloud && pkill -f gateway ; go build -o bin/gateway ./cmd/gateway && ./bin/gateway --config configs/gateway.yaml