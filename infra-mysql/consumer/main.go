package main

import (
    "database/sql"
    "fmt"
    "log"
    "time"

    "github.com/nsqio/go-nsq"
    _ "github.com/go-sql-driver/mysql"
)

const batchSize = 100

type MessageHandler struct {
    db        *sql.DB
    buffer    []map[string]interface{}
    lastSent  time.Time
    idCounter uint32 // Counter for generating unique IDs
}

func (h *MessageHandler) HandleMessage(m *nsq.Message) error {
    info := string(m.Body)
    if len(info) > 65535 {
        return fmt.Errorf("info data is too long: %d characters", len(info))
    }
    fmt.Println("Received message from NSQ:", info)
    _, err := h.db.Exec("INSERT INTO system_info (info) VALUES (?)", info)
    if err != nil {
        fmt.Println("Error inserting into MySQL:", err)
        return err
    }
    fmt.Println("Inserted into MySQL successfully")
    return nil
}

func connectToDatabase() (*sql.DB, error) {
    var db *sql.DB
    var err error
    for i := 0; i < 10; i++ { // Retry up to 10 times
        db, err = sql.Open("mysql", "root:Localhost@123@tcp(mysql:3306)/infra_monitoring")
        if err == nil {
            err = db.Ping()
            if err == nil {
                return db, nil
            }
        }
        log.Printf("Error connecting to database: %v. Retrying...", err)
        time.Sleep(5 * time.Second)
    }
    return nil, err
}

func main() {
    db, err := connectToDatabase()
    if err != nil {
        log.Fatal("Could not connect to the database:", err)
    }
    defer db.Close()
    fmt.Println("Connected to the database successfully")

    // Create table
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS system_info (
            id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
            info VARCHAR(255),
            cpu_usage FLOAT,
            memory_usage FLOAT,
            disk_usage TEXT,
            processes TEXT,
            connections TEXT,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
    `)
    if err != nil {
        log.Fatal("Error creating table in MySQL:", err)
    }
    fmt.Println("Table created successfully")

    handler := &MessageHandler{
        db:        db,
        buffer:    make([]map[string]interface{}, 0, batchSize),
        lastSent:  time.Now(),
        idCounter: 1, // Start ID counter
    }
    config := nsq.NewConfig()
    consumer, err := nsq.NewConsumer("system_info", "channel", config)
    if err != nil {
        log.Fatal("Error creating NSQ consumer:", err)
    }
    consumer.AddHandler(handler)

    err = consumer.ConnectToNSQD("nsq:4150")
    if err != nil {
        log.Fatal("Error connecting to NSQ:", err)
    }
    fmt.Println("Connected to NSQ successfully")

    <-consumer.StopChan
}
