package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "os/exec"
    "time"

    "github.com/shirou/gopsutil/cpu"
    "github.com/shirou/gopsutil/disk"
    "github.com/shirou/gopsutil/mem"
    "github.com/shirou/gopsutil/net"
    "github.com/shirou/gopsutil/process"
)

type DiskUsageInfo struct {
    Path              string  `json:"path"`
    Fstype            string  `json:"fstype"`
    Total             string  `json:"total"`
    Free              string  `json:"free"`
    Used              string  `json:"used"`
    UsedPercent       float64 `json:"usedPercent"`
    InodesTotal       uint64  `json:"inodesTotal"`
    InodesUsed        uint64  `json:"inodesUsed"`
    InodesFree        uint64  `json:"inodesFree"`
    InodesUsedPercent float64 `json:"inodesUsedPercent"`
}

func formatBytes(bytes uint64) string {
    const unit = 1024
    if bytes < unit {
        return fmt.Sprintf("%d B", bytes)
    }
    div, exp := int64(unit), 0
    for n := bytes / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func getSystemInfo() (string, float64, float64, []string, []string, map[string]DiskUsageInfo) {
    // Get general system info
    out, err := exec.Command("uname", "-a").Output()
    if err != nil {
        fmt.Println(err)
    }
    systemInfo := string(out)

    // Get CPU usage percentage
    cpuPercent, err := cpu.Percent(time.Second, false)
    if err != nil {
        fmt.Println("Error getting CPU usage:", err)
    }

    // Get memory usage
    memInfo, err := mem.VirtualMemory()
    if err != nil {
        fmt.Println("Error getting memory info:", err)
    }

    // Get process list
    processes, err := process.Processes()
    if err != nil {
        fmt.Println("Error getting processes:", err)
    }
    processList := []string{}
    for _, p := range processes {
        name, err := p.Name()
        if err == nil {
            processList = append(processList, name)
        }
    }

    // Get network connections
    connections, err := net.Connections("all")
    if err != nil {
        fmt.Println("Error getting network connections:", err)
    }
    connectionList := []string{}
    for _, conn := range connections {
        connectionList = append(connectionList, fmt.Sprintf("%s:%d %s:%d %s", conn.Laddr.IP, conn.Laddr.Port, conn.Raddr.IP, conn.Raddr.Port, conn.Status))
    }

    // Get disk usage
    partitions, err := disk.Partitions(true)
    if err != nil {
        fmt.Println("Error getting disk partitions:", err)
    }
    diskUsage := make(map[string]DiskUsageInfo)
    for _, partition := range partitions {
        usage, err := disk.Usage(partition.Mountpoint)
        if err != nil {
            fmt.Println("Error getting disk usage:", err)
        } else {
            diskUsage[partition.Mountpoint] = DiskUsageInfo{
                Path:              partition.Mountpoint,
                Fstype:            partition.Fstype,
                Total:             formatBytes(usage.Total),
                Free:              formatBytes(usage.Free),
                Used:              formatBytes(usage.Used),
                UsedPercent:       usage.UsedPercent,
                InodesTotal:       usage.InodesTotal,
                InodesUsed:        usage.InodesUsed,
                InodesFree:        usage.InodesFree,
                InodesUsedPercent: usage.InodesUsedPercent,
            }
        }
    }

    return systemInfo, cpuPercent[0], memInfo.UsedPercent, processList, connectionList, diskUsage
}

func sendInfo() {
    for {
        info, cpuUsage, memUsage, processes, connections, diskUsage := getSystemInfo()
        fmt.Printf("Collected System Info:\n%s\nCPU Usage: %.2f%%\nMemory Usage: %.2f%%\nDisk Usage: %v\nProcesses: %v\nConnections: %v\n", info, cpuUsage, memUsage, diskUsage, processes, connections)
        jsonData, err := json.Marshal(map[string]interface{}{
            "info":           info,
            "cpu_usage":      cpuUsage,
            "memory_usage":   memUsage,
            "disk_usage":     diskUsage,
            "processes":      processes,
            "connections":    connections,
        })
        if err != nil {
            fmt.Println("Error marshaling JSON:", err)
            continue
        }
        resp, err := http.Post("http://receiver:3000/systeminfo", "application/json", bytes.NewBuffer(jsonData))
        if err != nil {
            fmt.Println("Error sending info:", err)
        } else {
            fmt.Println("Info sent successfully, status code:", resp.StatusCode)
        }
        if resp != nil {
            resp.Body.Close()
        }
        time.Sleep(10 * time.Second)
    }
}

func main() {
    sendInfo()
}
