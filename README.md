# kubejobs

kubejobs is an http server that can concurrently schedule kubernetes jobs, based on the job priorities.


## Features

- Submit jobs to Kubernetes
- Manage job priorities
- Concurrent job schedule management

## Project Structure

```
kubejobs
├── cmd
│   └── server
│       └── main.go          # Entry point of the application
├── pkg
│   ├── api
│   │   ├── handlers
│   │   │   ├── processor.go # Job-related HTTP handlers
│   │   │   └── health.go    # Health check handler
│   │   │   
│   │   └── server.go        # HTTP server initialization
│   ├── kubernetes
│   │   ├── client.go        # Kubernetes client initialization
│   │   ├── jobs.go          # Job management functions
│   └── models
│       └── job.go           # JobSpec model definition
├── internal
│   ├── queue
│   │   └── priority_queue.go # Max-priority queue implementation
├── go.mod                    # Module definition and dependencies
├── go.sum                    # Dependency checksums
├── Dockerfile                # Docker image build instructions
├── Makefile                  # Build commands
└── README.md                 # Project documentation
```

## Setup Instructions

1. Clone the repository:
   ```
   git clone github.com/alwaysnewbie161/kube_jobs
   cd kubejobs
   ```

2. Install dependencies:
   ```
   go mod tidy
   ```

3. Build the application:
   ```
   make build
   ```

4. Run the application:
   ```
   make run
   ```

## API Endpoints

- `POST /jobs`: Schedule a job with priority to Kubernetes
- `GET /health`: Check the health status of the server
- `GET /jobs/pending`: Get current pending jobs in the queue
- `GET /jobs/running`: Get current in-flight jobs in the queue

## CLI Options

The server supports the following CLI options:

1. **`--kubeconfig <path>`**:
   - Specifies the path to the Kubernetes configuration file.
   - Default: `~/.kube/config`.

   Example:
   ```bash
   ./kubejobs --kubeconfig /path/to/kubeconfig
   ```

2. **`--port <port>`**:
   - Specifies the port on which the HTTP server will run.
   - Default: `8000`.

   Example:
   ```bash
   ./kubejobs --port 8080
   ```

3. **`--max-concurrency <N>`**:
   - Specifies the maximum number of concurrent job submissions.
   - Default: `2`.

   Example:
   ```bash
   ./kubejobs --max-concurrency 5
   ```

## Example Curl Requests

Here are some example curl requests to interact with the server:

1. **Submit a Job**:
   ```bash
   curl -X POST -H "Content-Type: application/json" -d '{"name":"job1","priority":2}' http://localhost:8000/jobs
   ```

2. **Check Pending Jobs**:
   ```bash
   curl http://localhost:8000/jobs/pending
   ```

3. **Check Running Jobs**:
   ```bash
   curl http://localhost:8000/jobs/running
   ```

4. **Health Check**:
   ```bash
   curl http://localhost:8000/health
   ```


## Example usage

This is the scheduling behaviour of a sample schedule of jobs on the server
For testing purposes, the server is run in dryrun mode that simulates 10second delay for every job
`Schedule timeline`:
1. 3 jobs are scheduled with priorities [2,5,5]
2. wait for 5 secs
3. 2 more jobs are scheduled with priorities [3,3]

```
Scheduled job job1: {"job_id":"95c713f8-4ba4-11f0-8000-000000000000","name":"job1","queue_position":"0","status":"Job submitted successfully"}

Scheduled job job2: {"job_id":"95c720f0-4ba4-11f0-8000-000000000000","name":"job2","queue_position":"0","status":"Job submitted successfully"}

Scheduled job job3: {"job_id":"95c72a78-4ba4-11f0-8000-000000000000","name":"job3","queue_position":"2","status":"Job submitted successfully"}

Scheduled job job4: {"job_id":"98c25a90-4ba4-11f0-8000-000000000000","name":"job4","queue_position":"0","status":"Job submitted successfully"}

Scheduled job job5: {"job_id":"98c264d6-4ba4-11f0-8000-000000000000","name":"job5","queue_position":"2","status":"Job submitted successfully"}

// after 5 seconds

Pending Jobs:
/jobs/pending Response: [{"name":"job1","job_id":"95c713f8-4ba4-11f0-8000-000000000000","priority":2},{"name":"job4","job_id":"98c25a90-4ba4-11f0-8000-000000000000","priority":3},{"name":"job5","job_id":"98c264d6-4ba4-11f0-8000-000000000000","priority":3}]

Running Jobs:
/jobs/running Response: [{"name":"job2","job_id":"95c720f0-4ba4-11f0-8000-000000000000","priority":5},{"name":"job3","job_id":"95c72a78-4ba4-11f0-8000-000000000000","priority":5}]


// after 5 seconds

Pending Jobs:
/jobs/pending Response: [{"name":"job1","job_id":"95c713f8-4ba4-11f0-8000-000000000000","priority":2},{"name":"job5","job_id":"98c264d6-4ba4-11f0-8000-000000000000","priority":3}]

Running Jobs:
/jobs/running Response: [{"name":"job4","job_id":"98c25a90-4ba4-11f0-8000-000000000000","priority":3},{"name":"job3","job_id":"95c72a78-4ba4-11f0-8000-000000000000","priority":5}]

// after 5 seconds

Pending Jobs:
/jobs/pending Response: [{"name":"job5","job_id":"98c264d6-4ba4-11f0-8000-000000000000","priority":3},{"name":"job1","job_id":"95c713f8-4ba4-11f0-8000-000000000000","priority":2}]

Running Jobs:
/jobs/running Response: [{"name":"job4","job_id":"98c25a90-4ba4-11f0-8000-000000000000","priority":3},{"name":"job3","job_id":"95c72a78-4ba4-11f0-8000-000000000000","priority":5}]

// after 5 seconds

Pending Jobs:
/jobs/pending Response: [{"name":"job1","job_id":"95c713f8-4ba4-11f0-8000-000000000000","priority":2}]

Running Jobs:
/jobs/running Response: [{"name":"job4","job_id":"98c25a90-4ba4-11f0-8000-000000000000","priority":3},{"name":"job5","job_id":"98c264d6-4ba4-11f0-8000-000000000000","priority":3}]

// after 5 seconds

Pending Jobs:
/jobs/pending Response: [{"name":"job1","job_id":"95c713f8-4ba4-11f0-8000-000000000000","priority":2}]

Running Jobs:
/jobs/running Response: [{"name":"job4","job_id":"98c25a90-4ba4-11f0-8000-000000000000","priority":3},{"name":"job5","job_id":"98c264d6-4ba4-11f0-8000-000000000000","priority":3}]

// after 5 seconds

Pending Jobs:
/jobs/pending Response: []

Running Jobs:
/jobs/running Response: [{"name":"job1","job_id":"95c713f8-4ba4-11f0-8000-000000000000","priority":2},{"name":"job5","job_id":"98c264d6-4ba4-11f0-8000-000000000000","priority":3}]

All jobs are processed successfully
```