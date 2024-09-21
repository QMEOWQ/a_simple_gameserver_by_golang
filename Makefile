gameclient:
	@go build -o bin/client client/main.go
	@./bin/client

gameserver:
	@go build -o bin/server server/main.go
	@./bin/server

