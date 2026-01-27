# Advanced Process Monitoring & Troubleshooting Guide

This guide outlines high-value dashboard visualizations and alerts for process-level monitoring. These recommendations are designed to solve specific software engineering problems such as memory leaks, thread starvation, I/O bottlenecks, and architecture regression.

## 1. Memory Health & Leak Detection

### Problem: Gradual Memory Leaks
**Scenario:** A background worker or API service slowly consumes more memory over days due to unclosed file handles, cached objects not being evicted, or goroutine leaks. This eventually leads to OOM (Out of Memory) kills and service disruption.

### Widget 1: Process Memory Usage History
**Visual:** Line Chart (Timeseries)
**Value:** Visualizes memory usage over time to help identify trends, such as stability (flat line) or potential leaks (continuous growth).
**NRQL:**
```sql
SELECT average(process.memory.usage) / 1024 / 1024 as 'Memory Usage (MB)' 
FROM Metric 
WHERE process.name IS NOT NULL 
FACET process.name 
TIMESERIES 1 hour 
SINCE 7 days ago
```

### Widget 2: Process Memory Stability
**Visual:** Table
**Value:** Lists processes with the highest fluctuation in memory usage, helping to identify unstable applications.
**NRQL:**
```sql
SELECT max(process.memory.usage) - min(process.memory.usage) as 'Usage Delta (Bytes)', 
       latest(process.memory.usage) as 'Current Usage' 
FROM Metric 
FACET process.name 
LIMIT 10
```

---

## 2. Concurrency & Threading Issues

### Problem: Thread Leaks & Deadlocks
**Scenario:** An application creates threads for incoming requests but fails to terminate them correctly (e.g., waiting on a blocked channel or deadlocked mutex). Thread count rises until the OS limit is reached (`ulimit`), causing the app to crash or hang.

### Widget 3: Thread Count vs CPU Utilization
**Visual:** Area Chart
**Value:** Correlates thread count with CPU usage. High thread counts with low CPU activity may indicate deadlocks, while high threads with high CPU may indicate a thread leak or excessive processing.
**NRQL:**
```sql
SELECT average(process.threads) as 'Thread Count', 
       average(process.cpu.utilization) * 100 as 'CPU %' 
FROM Metric 
WHERE process.name = 'java' OR process.name = 'node' 
FACET process.name 
TIMESERIES
```

---

## 3. CPU Efficiency & Starvation

### Problem: "Noisy Neighbor" & CPU Throttling
**Scenario:** In a containerized or shared environment, one process (e.g., a backup agent or log compressor) spikes CPU, starving critical business applications.

### Widget 4: CPU Usage by Process
**Visual:** Pie Chart or Stacked Bar
**Value:** Shows exactly which process is consuming the host's CPU cycles relative to others.
**NRQL:**
```sql
SELECT sum(process.cpu.utilization) 
FROM Metric 
FACET process.name 
LIMIT 10 
SINCE 30 minutes ago
```

### Widget 5: Context Switch Rate
**Scenario:** High context switching indicates the CPU is thrashing between tasks rather than doing work, often due to excessive locking contention or too many active threads.
**Visual:** Timeseries
**NRQL:**
```sql
SELECT rate(avg(process.context_switches), 1 minute) 
FROM Metric 
FACET process.name 
TIMESERIES
```
*(Note: Requires `process.context_switches` metric to be enabled via hostmetrics receiver config)*

---

## 4. I/O Performance & Bottlenecks

### Problem: Disk Thrashing
**Scenario:** A database or logging sidecar writes to disk so aggressively that it fills the OS page cache or saturates IOPS, causing high wait times for the main application.

### Widget 6: Disk I/O Volume by Process
**Visual:** Bar Chart
**Value:** Instantly identifies which process is performing the most disk I/O operations.
**NRQL:**
```sql
SELECT average(process.disk.io.read_bytes + process.disk.io.write_bytes) / 1024 as 'Avg I/O (KB/s)' 
FROM Metric 
FACET process.name 
LIMIT 10 
TIMESERIES
```

---

## 5. Stability & Crash Loops

### Problem: Silent Crash Loops
**Scenario:** A process crashes and is immediately restarted by systemd or k8s. Service appears "up" but is constantly cold-starting, leading to poor performance and dropped requests during the restart window.

### Widget 7: Recently Started Processes
**Visual:** Table
**Value:** Highlights processes that have recently started. Recurring entries for the same service may indicate a crash loop.
**NRQL:**
```sql
SELECT latest(process.uptime) as 'Uptime (Seconds)' 
FROM Metric 
FACET process.name, host.name 
WHERE process.uptime < 300 
LIMIT 20
```
*(Alert Condition Suggestion: Alert if `process.uptime` resets more than 3 times in 1 hour)*

---

## 6. File Descriptor Exhaustion

### Problem: "Too many open files"
**Scenario:** Sockets or files are opened but not closed. Eventually, the process hits the limit and starts failing all network connections or file writes.

### Widget 8: Open File Descriptors
**Visual:** Line Chart (Timeseries)
**Value:** Monitoring `process.open_file_descriptors` allows you to track usage trends and avoid exhaustion limits.
**NRQL:**
```sql
SELECT average(process.open_file_descriptors) 
FROM Metric 
FACET process.name 
TIMESERIES
```
*(Note: Requires `process.open_file_descriptors` metric to be enabled)*

---

## Summary of Key Metrics to Enable

To power these dashboards, ensure your OpenTelemetry `hostmetrics` receiver is configured to scrape the following:

| Metric | Essential for |
|--------|---------------|
| `process.memory.usage` | Memory Leaks |
| `process.cpu.utilization` | CPU Starvation |
| `process.threads` | Deadlocks / Leaks |
| `process.disk.io` | I/O Bottlenecks |
| `process.uptime` | Crash Loops |
| `process.open_file_descriptors` | Connection Leaks |

