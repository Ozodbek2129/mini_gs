FROM golang:1.23.4 AS builder 

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main cmd/main.go

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y libc6 && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/main .

COPY --from=builder /app/datas.json  ./datas.json
COPY --from=builder /app/micro_gs.json  ./micro_gs.json 
COPY --from=builder /app/micro_gs1.json  ./micro_gs1.json 
COPY --from=builder /app/minigs12.json  ./minigs12.json 
COPY --from=builder /app/booling.json  ./booling.json 
COPY --from=builder /app/email/template.html ./email/
COPY --from=builder /app/.env .

CMD ["./main"]