package jobs

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"secshell/colors"
	"secshell/logging"
	"secshell/ui/gui"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/process"
)

var clockTicks float64

func init() {
	// Most modern Linux systems use 100 Hz
	clockTicks = 100.0
}

type Job struct {
	ID          int
	Command     string
	Status      string
	StartTime   time.Time
	EndTime     time.Time
	ExitCode    int
	CPU         float64
	Memory      float64 // in MB
	ThreadCount int
	Process     *os.Process // Store the process
	sync.Mutex
}

func DrawTable(header string, data []string, bg_color string) {
	cmdArgs := []string{"table", header}
	cmdArgs = append(cmdArgs, data...)
	cmdArgs = append(cmdArgs, "hollow", bg_color)

	cmdTable := exec.Command("drawbox", cmdArgs...)
	cmdTable.Stdout = os.Stdout
	cmdTable.Stderr = os.Stderr
	if err := cmdTable.Run(); err != nil {
		logging.LogError(err)
		fmt.Println("Error executing drawbox table command:", err)
	}

}

// listJobs lists all jobs
func ListJobs(jobs map[int]*Job) {

	if len(jobs) == 0 {
		fmt.Println("No jobs found.")
		return
	}

	var services []string
	for pid, job := range jobs {
		job.Lock()
		updateJobStats(job)
		status := job.Status
		command := job.Command
		cpu := fmt.Sprintf("%.1f%%", job.CPU)
		mem := fmt.Sprintf("%.1fMB", job.Memory)
		threads := strconv.Itoa(job.ThreadCount)

		// Choose color based on status
		statusColor := colors.Red
		if status == "running" {
			statusColor = colors.Green
		}

		job.Unlock()

		entry := fmt.Sprintf("%d,%s,%s%s%s,%s,%s,%s",
			pid, command, statusColor, status, colors.Reset,
			cpu, mem, threads)
		services = append(services, entry)
	}

	DrawTable("PID,COMMAND,STATUS,CPU,MEMORY,THREADS", services, "bold_white")

}

func AddJob(jobs map[int]*Job, pid int, command string, process *os.Process) {
	job := &Job{
		ID:        pid,
		Command:   command,
		Status:    "running",
		StartTime: time.Now(),
		Process:   process,
	}
	jobs[pid] = job
	logging.LogAlert(fmt.Sprintf("Job [%d] added: %s", pid, command))
	gui.AlertBox(fmt.Sprintf("[%d] %s running in background", pid, command))
}

// RemoveJob removes a job from the jobs map
func RemoveJob(jobs map[int]*Job, pid int) {
	delete(jobs, pid)
	logging.LogAlert(fmt.Sprintf("Job [%d] removed", pid))
	gui.AlertBox(fmt.Sprintf("Job [%d] removed", pid))
}

func StopJob(jobs map[int]*Job, pid int) {
	job, ok := jobs[pid]
	if !ok {
		logging.LogAlert(fmt.Sprintf("No such job: %d", pid))
		gui.ErrorBox(fmt.Sprintf("No such job: %d", pid))
		return
	}

	job.Lock()
	defer job.Unlock()

	if job.Status != "running" {
		logging.LogAlert(fmt.Sprintf("Job [%d] is not running", pid))
		gui.AlertBox(fmt.Sprintf("Job [%d] is not running", pid))
		return
	}

	// Send interrupt signal to the process
	if err := job.Process.Signal(os.Interrupt); err != nil {
		logging.LogError(err)
		gui.ErrorBox(fmt.Sprintf("Failed to stop job [%d]: %s", pid, err))
		return
	}

	job.Status = "stopped"
	logging.LogAlert(fmt.Sprintf("Job [%d] stopped", pid))
	gui.AlertBox(fmt.Sprintf("Job [%d] stopped", pid))
}

func StopJobClean(jobs map[int]*Job, pid int) string {
	job, ok := jobs[pid]
	if !ok {
		logging.LogAlert(fmt.Sprintf("No such job: %d", pid))
		return fmt.Sprintf("No such job: %d", pid)

	}

	job.Lock()
	defer job.Unlock()

	if job.Status != "running" {
		logging.LogAlert(fmt.Sprintf("Job [%d] is not running", pid))
		return fmt.Sprintf("Job [%d] is not running", pid)

	}

	// Send interrupt signal to the process
	if err := job.Process.Signal(os.Interrupt); err != nil {
		logging.LogError(err)
		return fmt.Sprintf("Failed to stop job [%d]: %s", pid, err)

	}

	job.Status = "stopped"
	logging.LogAlert(fmt.Sprintf("Job [%d] stopped", pid))
	return fmt.Sprintf("Job [%d] stopped", pid)
}

func GetJobStatus(jobs map[int]*Job, pid int) {
	job, ok := jobs[pid]
	if !ok {
		logging.LogAlert(fmt.Sprintf("No such job: %d", pid))
		gui.ErrorBox(fmt.Sprintf("No such job: %d", pid))
		return
	}

	job.Lock()
	updateJobStats(job)
	defer job.Unlock()

	fmt.Printf("Job [%d] Details:\n", pid)
	fmt.Printf("  Status: %s\n", job.Status)
	fmt.Printf("  Command: %s\n", job.Command)
	fmt.Printf("  CPU Usage: %.1f%%\n", job.CPU)
	fmt.Printf("  Memory Usage: %.1f MB\n", job.Memory)
	fmt.Printf("  Threads: %d\n", job.ThreadCount)
	fmt.Printf("  Running since: %s\n", job.StartTime.Format(time.RFC1123))
}

func StartJob(jobs map[int]*Job, pid int) {
	job, ok := jobs[pid]
	if !ok {
		logging.LogAlert(fmt.Sprintf("No such job: %d", pid))
		gui.ErrorBox(fmt.Sprintf("No such job: %d", pid))
		return
	}

	job.Lock()
	defer job.Unlock()

	if job.Status == "running" {
		logging.LogAlert(fmt.Sprintf("Job [%d] is already running", pid))
		gui.AlertBox(fmt.Sprintf("Job [%d] is already running", pid))
		return
	}

	// Start the process
	err := job.Process.Signal(syscall.SIGCONT)
	if err != nil {
		logging.LogError(err)
		gui.ErrorBox(fmt.Sprintf("Failed to start job [%d]: %s", pid, err))
		return
	}

	job.Status = "running"
	logging.LogAlert(fmt.Sprintf("Job [%d] started", pid))
	gui.AlertBox(fmt.Sprintf("Job [%d] started", pid))
}

func StartJobClean(jobs map[int]*Job, pid int) string {
	job, ok := jobs[pid]
	if !ok {
		logging.LogAlert(fmt.Sprintf("No such job: %d", pid))
		return fmt.Sprintf("No such job: %d", pid)
	}

	job.Lock()
	defer job.Unlock()

	if job.Status == "running" {
		logging.LogAlert(fmt.Sprintf("Job [%d] is already running", pid))
		return fmt.Sprintf("Job [%d] is already running", pid)
	}

	// Start the process
	err := job.Process.Signal(syscall.SIGCONT)
	if err != nil {
		logging.LogError(err)
		return fmt.Sprintf("Failed to start job [%d]: %s", pid, err)
	}

	job.Status = "running"
	logging.LogAlert(fmt.Sprintf("Job [%d] started", pid))
	return fmt.Sprintf("Job [%d] started", pid)
}

func ClearFinishedJobs(jobs map[int]*Job) {
	gui.TitleBox("Clear Finished Jobs")

	if len(jobs) == 0 {
		fmt.Println("No jobs found.")
		return
	}

	for pid, job := range jobs {
		job.Lock()
		status := job.Status
		job.Unlock()
		if status != "running" {
			RemoveJob(jobs, pid)
		}
	}
}

func RunJobsCommand(action string, pid int, jobs map[int]*Job) {

	switch action {
	case "stop":
		gui.TitleBox("Stop Job")
		StopJob(jobs, pid)
	case "status":
		gui.TitleBox("Job Status")
		GetJobStatus(jobs, pid)
	case "start":
		gui.TitleBox("Start Job")
		StartJob(jobs, pid)
	case "list":
		gui.TitleBox("List Jobs")
		ListJobs(jobs)
	case "clear-finished":
		ClearFinishedJobs(jobs)
	case "--help", "-h", "help":
		ShowHelp()
	default:
		gui.ErrorBox("Invalid action. Use stop, status, start, list, or clear-finished.")
	}
}

func ShowHelp() {
	gui.TitleBox("Jobs Help")
	fmt.Println("Usage: jobs <action> [PID]")
	fmt.Println("Actions:")
	fmt.Println("  list           - Lists all jobs")
	fmt.Println("  stop <PID>     - Stops the job with the specified PID")
	fmt.Println("  start <PID>    - Starts/resumes the job with the specified PID")
	fmt.Println("  status <PID>   - Shows the status of the job with the specified PID")
	fmt.Println("  clear-finished - Clears all finished jobs from the list")
}

// Use a map to cache CPU stats between calls for better delta calculations
var (
	previousCPUTimes     map[int]float64   = make(map[int]float64)
	previousCPUTimestamp map[int]time.Time = make(map[int]time.Time)
)

func updateJobStats(job *Job) {
	// Get the process object using gopsutil
	p, err := process.NewProcess(int32(job.Process.Pid))
	if err != nil {
		// Process may have terminated or cannot be accessed
		job.Status = "terminated"
		return
	}

	// Create timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Get system CPU info first
	systemCPU, err := cpu.InfoWithContext(ctx)
	numCPUs := 1
	if err == nil && len(systemCPU) > 0 {
		// Count logical cores for accurate CPU percentage calculation
		numCPUs = len(systemCPU)
	}

	// Get memory info
	memInfo, err := p.MemoryInfoWithContext(ctx)
	if err == nil && memInfo != nil {
		// Convert RSS from bytes to MB
		job.Memory = float64(memInfo.RSS) / (1024 * 1024)
	}

	// Get thread count
	numThreads, err := p.NumThreadsWithContext(ctx)
	if err == nil {
		job.ThreadCount = int(numThreads)
	}

	// Check process status
	status, err := p.StatusWithContext(ctx)
	if err == nil {
		statusStr := strings.Join(status, "")
		if strings.Contains(statusStr, "stop") {
			job.Status = "stopped"
		} else {
			job.Status = "running"
		}
	}

	// Get CPU time using Times() which returns concrete numbers
	times, err := p.TimesWithContext(ctx)
	now := time.Now()

	if err == nil && times != nil {
		// Instead of using the deprecated Total() method, calculate total time manually
		currentCPUTime := times.User + times.System + times.Iowait + times.Nice + times.Irq +
			times.Softirq + times.Steal + times.Guest + times.GuestNice

		// Calculate CPU usage as a percentage
		if prevTime, hasPrev := previousCPUTimes[job.ID]; hasPrev {
			prevTimestamp := previousCPUTimestamp[job.ID]
			timeDelta := now.Sub(prevTimestamp).Seconds()

			if timeDelta > 0 {
				// Calculate CPU usage as change in CPU time divided by elapsed time
				cpuDelta := currentCPUTime - prevTime
				cpuPercentage := (cpuDelta / timeDelta) * 100.0

				// Adjust for multi-core systems
				if numCPUs > 1 {
					// Show percentage relative to all cores
					cpuPercentage = cpuPercentage * float64(numCPUs)
				}

				// Normalize CPU percentage (shouldn't exceed 100% per core)
				if cpuPercentage > float64(100*numCPUs) {
					cpuPercentage = float64(100 * numCPUs)
				}

				// Store the CPU usage
				job.CPU = cpuPercentage
			}
		}

		// Update previous values for next calculation
		previousCPUTimes[job.ID] = currentCPUTime
		previousCPUTimestamp[job.ID] = now

		// If this is the first reading, try to get an immediate value
		if job.CPU <= 0.1 {
			// Try to get percent directly for first reading
			if percent, err := p.CPUPercentWithContext(ctx); err == nil && percent > 0 {
				job.CPU = percent
			} else {
				// Use system CPU percentage to estimate if all else fails
				systemPercent, _ := cpu.PercentWithContext(ctx, 100*time.Millisecond, false)
				if len(systemPercent) > 0 {
					// Just assign a small percentage to show activity
					job.CPU = systemPercent[0] / float64(numCPUs) / 4
					if job.CPU < 0.1 {
						job.CPU = 0.0 // Set minimum visible CPU usage for running processes
					}
				}
			}
		}
	}
}
