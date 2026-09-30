package model

type Job struct {
	JobID       string `json:"job_id"`
	Description string `json:"description"`
	Schedule    string `json:"schedule"`
	Task        Task   `json:"task"`
}

type Task struct {
	Type    string `json:"type"`
	Command string `json:"command,omitempty"`
}
