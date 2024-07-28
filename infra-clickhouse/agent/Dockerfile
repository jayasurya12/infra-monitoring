FROM golang:1.18
WORKDIR /app

# Copy application source code
COPY . .

# Initialize Go module and download dependencies
RUN go mod init agent && \
    go get github.com/shirou/gopsutil/cpu && \
    go get github.com/shirou/gopsutil/mem && \
    go get github.com/shirou/gopsutil/net && \
    go get github.com/shirou/gopsutil/process && \
    go get github.com/shirou/gopsutil/disk && \  
    go mod tidy && \
    go build -o main .

# Run the application
CMD ["./main"]
