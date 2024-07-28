FROM golang:1.18
WORKDIR /app
COPY . .
RUN go mod init agent && \
    go get github.com/shirou/gopsutil/cpu && \
    go get github.com/shirou/gopsutil/mem && \
    go get github.com/shirou/gopsutil/net && \
    go get github.com/shirou/gopsutil/process && \
    go get github.com/shirou/gopsutil/disk && \  
    go mod tidy && \
    go build -o main .
    
CMD ["./main"]
