package jobs

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"secshell/colors"
	"secshell/drawbox"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
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
	drawbox.PrintAlert(fmt.Sprintf("[%d] %s running in background", pid, command))
}

// RemoveJob removes a job from the jobs map
func RemoveJob(jobs map[int]*Job, pid int) {
	delete(jobs, pid)
	drawbox.PrintAlert(fmt.Sprintf("Job [%d] removed", pid))
}

func StopJob(jobs map[int]*Job, pid int) {
	job, ok := jobs[pid]
	if !ok {
		drawbox.PrintError(fmt.Sprintf("No such job: %d", pid))
		return
	}

	job.Lock()
	defer job.Unlock()

	if job.Status != "running" {
		drawbox.PrintAlert(fmt.Sprintf("Job [%d] is not running", pid))
		return
	}

	// Send interrupt signal to the process
	if err := job.Process.Signal(os.Interrupt); err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to stop job [%d]: %s", pid, err))
		return
	}

	job.Status = "stopped"
	drawbox.PrintAlert(fmt.Sprintf("Job [%d] stopped", pid))
}

func GetJobStatus(jobs map[int]*Job, pid int) {
	job, ok := jobs[pid]
	if !ok {
		drawbox.PrintError(fmt.Sprintf("No such job: %d", pid))
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
		drawbox.PrintError(fmt.Sprintf("No such job: %d", pid))
		return
	}

	job.Lock()
	defer job.Unlock()

	if job.Status == "running" {
		drawbox.PrintAlert(fmt.Sprintf("Job [%d] is already running", pid))
		return
	}

	// Start the process
	err := job.Process.Signal(syscall.SIGCONT)
	if err != nil {
		drawbox.PrintError(fmt.Sprintf("Failed to start job [%d]: %s", pid, err))
		return
	}

	job.Status = "running"
	drawbox.PrintAlert(fmt.Sprintf("Job [%d] started", pid))
}

func ClearFinishedJobs(jobs map[int]*Job) {
	drawbox.RunDrawbox("Clear Finished Jobs", "bold_white")

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
	drawbox.RunDrawbox("Jobs", "bold_white")

	switch action {
	case "stop":
		StopJob(jobs, pid)
	case "status":
		GetJobStatus(jobs, pid)
	case "start":
		StartJob(jobs, pid)
	case "list":
		ListJobs(jobs)
	case "clear-finished":
		ClearFinishedJobs(jobs)
	case "--help", "-h", "help":
		ShowHelp()
	default:
		drawbox.PrintError("Invalid action. Use stop, status, start, list, or clear-finished.")
	}
}

func ShowHelp() {
	drawbox.RunDrawbox("Jobs Help", "bold_white")
	fmt.Println("Usage: jobs <action> [PID]")
	fmt.Println("Actions:")
	fmt.Println("  list           - Lists all jobs")
	fmt.Println("  stop <PID>     - Stops the job with the specified PID")
	fmt.Println("  start <PID>    - Starts/resumes the job with the specified PID")
	fmt.Println("  status <PID>   - Shows the status of the job with the specified PID")
	fmt.Println("  clear-finished - Clears all finished jobs from the list")
}

func updateJobStats(job *Job) {
	procPath := fmt.Sprintf("/proc/%d", job.Process.Pid)

	// Read memory info
	if statm, err := os.ReadFile(fmt.Sprintf("%s/statm", procPath)); err == nil {
		fields := strings.Fields(string(statm))
		if len(fields) > 1 {
			if rss, err := strconv.ParseFloat(fields[1], 64); err == nil {
				job.Memory = rss * 4.0 / 1024 // Convert to MB
			}
		}
	}

	// Read CPU usage and thread count
	if stat, err := os.ReadFile(fmt.Sprintf("%s/stat", procPath)); err == nil {
		fields := strings.Fields(string(stat))
		if len(fields) > 22 {
			utime, _ := strconv.ParseFloat(fields[13], 64)
			stime, _ := strconv.ParseFloat(fields[14], 64)
			starttime, _ := strconv.ParseFloat(fields[21], 64)

			uptimeData, err := os.ReadFile("/proc/uptime")
			if err == nil {
				uptimeFields := strings.Fields(string(uptimeData))
				if len(uptimeFields) > 0 {
					systemUptime, _ := strconv.ParseFloat(uptimeFields[0], 64)
					totalTime := utime + stime
					elapsed := systemUptime - (starttime / clockTicks)

					if elapsed > 0 {
						// Calculate CPU usage without dividing by number of CPUs first
						cpuUsage := (totalTime / clockTicks / elapsed) * 100

						// Only divide by number of CPUs if the usage is significant
						if cpuUsage > 0.01 {
							numCPU := float64(runtime.NumCPU())
							job.CPU = cpuUsage / numCPU
						} else {
							// For very small usage, don't divide to avoid returning 0
							job.CPU = cpuUsage
						}
					}
				}
			}

			if threads, err := strconv.Atoi(fields[19]); err == nil {
				job.ThreadCount = threads
			}
		}
	}
}
