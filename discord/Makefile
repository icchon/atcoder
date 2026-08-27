NAME := discord

all: depend seed test ${NAME}

depend: 
	go mod tidy

run:
	go build -o ${NAME} ./cmd/server/main.go
	./${NAME}
	
seed:
	go run ./cmd/seed/main.go

test:
	go test -v ./...

${NAME}:
	go build -o ${NAME} ./cmd/server/main.go

.PHONY: all depend seed test
