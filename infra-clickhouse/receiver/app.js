const express = require('express');
const bodyParser = require('body-parser');
const nsq = require('nsqjs');

const app = express();
app.use(bodyParser.json());

const writer = new nsq.Writer('nsq', 4150);
writer.connect();

app.post('/systeminfo', (req, res) => {
  const { info, cpu_usage, memory_usage, disk_usage, processes, connections } = req.body;

  // Log received data
  console.log("Received System Info:", info);
  console.log("CPU Usage:", cpu_usage);
  console.log("Memory Usage:", memory_usage);
  console.log("Disk Usage:", disk_usage);
  console.log("Processes:", processes);
  console.log("Connections:", connections);

  // Publish data to NSQ
  writer.publish('system_info', JSON.stringify({
    info,
    cpu_usage,
    memory_usage,
    disk_usage,
    processes,
    connections
  }), (err) => {
    if (err) {
      console.error("Error publishing to NSQ:", err);
    } else {
      console.log("Published to NSQ successfully");
    }
  });

  res.send('Received');
});

app.listen(3000, () => {
  console.log('Receiver listening on port 3000');
});
