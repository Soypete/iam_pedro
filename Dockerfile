FROM golang:1.23-alpine

ENV OPENAI_API_KEY="test"

RUN apk add --no-cache git

WORKDIR /app
COPY go.* ./
RUN go mod download

EXPOSE 6060

COPY . ./
RUN go build -v -o main .

CMD ["/app/main", "-model", "meta-llama3.1"]
