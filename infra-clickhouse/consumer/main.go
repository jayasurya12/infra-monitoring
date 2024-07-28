package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "strings"
    "time"

    "github.com/nsqio/go-nsq"
    _ "github.com/ClickHouse/clickhouse-go"
)

const batchSize = 100

type MessageHandler struct {
    db        *sql.DB
    buffer    []map[string]interface{}
    lastSent  time.Time
    idCounter uint32 // Counter for generating unique IDs
}

func (h *MessageHandler) batchInsert() error {
    if len(h.buffer) == 0 {
        return nil
    }

    // Open a transaction
    tx, err := h.db.Begin()
    if err != nil {
        return fmt.Errorf("error starting transaction: %w", err)
    }

    // Construct batch insert query
    var query strings.Builder
    query.WriteString("INSERT INTO infra_monitoring.system_info (id, info, cpu_usage, memory_usage, disk_usage, processes, connections, created_at) VALUES ")

    for i, msg := range h.buffer {
        // Extract fields from message
        id := h.idCounter
        info, ok := msg["info"].(string)
        if !ok {
            log.Printf("Invalid info field: %v", msg["info"])
            continue
        }
        cpuUsage, ok := msg["cpu_usage"].(float64)
        if !ok {
            log.Printf("Invalid cpu_usage field: %v", msg["cpu_usage"])
            continue
        }
        memoryUsage, ok := msg["memory_usage"].(float64)
        if !ok {
            log.Printf("Invalid memory_usage field: %v", msg["memory_usage"])
            continue
        }
        diskUsage, ok := msg["disk_usage"].(map[string]interface{})
        if !ok {
            log.Printf("Invalid disk_usage field: %v", msg["disk_usage"])
            continue
        }
        processes, ok := msg["processes"].([]interface{})
        if !ok {
            log.Printf("Invalid processes field: %v", msg["processes"])
            continue
        }
        connections, ok := msg["connections"].([]interface{})
        if !ok {
            log.Printf("Invalid connections field: %v", msg["connections"])
            continue
        }

        // Convert slices to JSON strings for ClickHouse insertion
        diskUsageJSON, _ := json.Marshal(diskUsage)
        processesJSON, _ := json.Marshal(processes)
        connectionsJSON, _ := json.Marshal(connections)

        // Append values to query
        query.WriteString(fmt.Sprintf("(%d, '%s', %f, %f, '%s', '%s', '%s', now())",
            id,
            strings.ReplaceAll(info, "'", "\\'"), // Escape single quotes
            cpuUsage,
            memoryUsage,
            strings.ReplaceAll(string(diskUsageJSON), "'", "\\'"), // Escape single quotes in JSON
            strings.ReplaceAll(string(processesJSON), "'", "\\'"), // Escape single quotes in JSON
            strings.ReplaceAll(string(connectionsJSON), "'", "\\'"), // Escape single quotes in JSON
        ))

        // Add a comma if it's not the last row
        if i < len(h.buffer)-1 {
            query.WriteString(", ")
        }
        h.idCounter++
    }

    // Finalize query
    queryStr := query.String()

    // Print the query for debugging
    fmt.Println("Executing query:", queryStr)

    // Execute the batch insert
    _, err = tx.Exec(queryStr)
    if err != nil {
        tx.Rollback()
        return fmt.Errorf("error inserting into ClickHouse: %w", err)
    }

    // Commit the transaction
    err = tx.Commit()
    if err != nil {
        return fmt.Errorf("error committing transaction: %w", err)
    }

    // Print success message
    fmt.Printf("Data inserted successfully, %d rows\n", len(h.buffer))

    // Clear the buffer
    h.buffer = nil
    return nil
}

func (h *MessageHandler) HandleMessage(m *nsq.Message) error {
    var msg map[string]interface{}
    err := json.Unmarshal(m.Body, &msg)
    if err != nil {
        fmt.Println("Error unmarshaling message:", err)
        return err
    }
    fmt.Println("Received message from NSQ:", msg)

    // Add message to buffer
    h.buffer = append(h.buffer, msg)

    // Batch insert if buffer size reaches batchSize or 1 minute has passed
    if len(h.buffer) >= batchSize || time.Since(h.lastSent) > time.Minute {
        if err := h.batchInsert(); err != nil {
            fmt.Println("Error during batch insert:", err)
            return err
        }
        h.lastSent = time.Now()
    }

    return nil
}

func connectToDatabase() (*sql.DB, error) {
    var db *sql.DB
    var err error
    dsn := "tcp://clickhouse:9000?database=infra_monitoring&username=default&password="
    for i := 0; i < 10; i++ { // Retry up to 10 times
        db, err = sql.Open("clickhouse", dsn)
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

    // Create database
    _, err = db.Exec("CREATE DATABASE IF NOT EXISTS infra_monitoring")
    if err != nil {
        log.Fatal("Error creating database in ClickHouse:", err)
    }
    fmt.Println("Database created successfully")

    // Create table
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS infra_monitoring.system_info (
            id UInt32,
            info String,
            cpu_usage Float64,
            memory_usage Float64,
            disk_usage String,
            processes Array(String),
            connections Array(String),
            created_at DateTime DEFAULT now()
        ) ENGINE = MergeTree()
        ORDER BY id
    `)
    if err != nil {
        log.Fatal("Error creating table in ClickHouse:", err)
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
