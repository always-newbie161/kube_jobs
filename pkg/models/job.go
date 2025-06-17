package models

// JobSpec represents a job specification submitted by the user
type JobSpec struct {
	ID        string   `json:"id,omitempty"`
	Name      string   `json:"name"`
	Priority  int      `json:"priority"`
	Command   string   `json:"command"`
	Args      []string `json:"args,omitempty"`
	Namespace string   `json:"namespace,omitempty"`
	Image     string   `json:"image"`
}

// JobResponse represents a simplified job response for API endpoints
type JobResponse struct {
	Name     string `json:"name"`
	JobID    string `json:"job_id,omitempty"`
	Priority int    `json:"priority"`
	Status   string `json:"status,omitempty"`
}
