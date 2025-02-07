FROM golang:1.23-bullseye
WORKDIR /app
ENV UUID=""
RUN mkdir -p /.cache && \
    chmod -R 777 /.cache
COPY api/ ./api/
COPY go.mod go.sum ./
RUN go mod download
RUN uname -a
EXPOSE 7860
CMD ["go", "run", "api/chat.go"]