package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
	"fmt"

	rundeck "github.com/lusis/go-rundeck/pkg/rundeck"
	"go.uber.org/zap"
)

type (

	// Timestamp is a helper for (un)marhalling time
	Timestamp time.Time

	// HookMessage is the message we receive from Alertmanager
	HookMessage struct {
		Version           string            `json:"version"`
		GroupKey          string            `json:"groupKey"`
		Status            string            `json:"status"`
		Receiver          string            `json:"receiver"`
		GroupLabels       map[string]string `json:"groupLabels"`
		CommonLabels      map[string]string `json:"commonLabels"`
		CommonAnnotations map[string]string `json:"commonAnnotations"`
		ExternalURL       string            `json:"externalURL"`
		Alerts            []Alert           `json:"alerts"`
	}

	// Alert is a single alert.
	Alert struct {
		Labels      map[string]string `json:"labels"`
		Annotations map[string]string `json:"annotations"`
		StartsAt    string            `json:"startsAt,omitempty"`
		EndsAt      string            `json:"EndsAt,omitempty"`
	}

	// just an example alert store. in a real hook, you would do something useful
	alertStore struct {
		rundeckClient *rundeck.Client
		secretToken   string
		logger        *zap.Logger
	}
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting logger: %v\n", err)
		os.Exit(1)
	}

	rundeckToken := os.Getenv("RUNDECK_TOKEN")
	if rundeckToken == "" {
		logger.Fatal("Rundeck token required. Cannot continue")
	}
	alcidesToken := os.Getenv("ALCIDES_TOKEN")
	if alcidesToken == "" {
		logger.Fatal("Alcides token not specified. Refusing to continue")
	}

	rundeckClient, err := rundeck.NewClientFromEnv()
	if err != nil {
		logger.Fatal("Rundeck broke due to: %s", zap.Error(err))
		return
	}

	s := &alertStore{
		rundeckClient: rundeckClient,
		secretToken:   alcidesToken,
		logger:        logger,
	}

	http.HandleFunc("/alerts", s.alertsHandler)
	http.HandleFunc("/status", s.statusHandler)

	logger.Info("[INFO] Server listening on :8080")
	if err := http.ListenAndServe(":"+"8080", nil); err != nil {
		logger.Fatal("[ERROR] %s", zap.Error(err))
	}
}

func (s *alertStore) alertsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.postHandler(w, r)
	default:
		http.Error(w, "unsupported HTTP method", 400)
	}
}

func (s *alertStore) statusHandler(w http.ResponseWriter, r *http.Request) {
	_, err := s.rundeckClient.Get("metrics/ping")
	if err != nil {
		s.logger.Error("Error w/ contacting rundeck.", zap.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s.logger.Info("Status checks OK")

	w.Header().Add("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Status checks OK"))
}

func (s *alertStore) postHandler(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	defer r.Body.Close()

	var m HookMessage
	if err := dec.Decode(&m); err != nil {
		s.logger.Error("error decoding message", zap.Error(err))
		http.Error(w, "invalid request body", 400)
		return
	}

	auth := r.Header.Get("Authorization")

	if s.isAuthed(auth) {
		for _, alert := range m.Alerts {
			if rundeckId, hasJob := alert.Labels["rundeck_job_id"]; hasJob {
				if argString, hasOpts := alert.Annotations["rundeck_args"]; hasOpts {
					s.logger.Info("Triggering rundeck job", zap.Any("job-id", rundeckId), zap.Any("job-params", argString))
					_, err := s.rundeckClient.RunJob(rundeckId, rundeck.RunJobArgs(argString))
					if err != nil {
						s.logger.Error("[ERROR] in rundeck job %s", zap.Error(err))
					}
				} else {
					s.logger.Info("Triggering rundeck job", zap.Any("job-id", rundeckId))
					_, err := s.rundeckClient.RunJob(rundeckId)
					if err != nil {
						s.logger.Error("[ERROR] in rundeck job %s", zap.Error(err))
					}
				}
			}
		}
	}
}

func (s *alertStore) isAuthed(authHeader string) bool {
	splitToken := strings.Split(authHeader, "Basic")
	if len(splitToken) != 2 {
		s.logger.Error("Auth header has no Basic", zap.Any("auth:", authHeader))
		return false
	}

	authHeader = strings.TrimSpace(splitToken[1])

	decoded, err := base64.StdEncoding.DecodeString(authHeader)
	if err != nil {
		s.logger.Error("[ERROR] %s", zap.Error(err))
	}

	authHeader = string(decoded)

	splitToken = strings.Split(authHeader, ":")
	if len(splitToken) != 2 {
		s.logger.Error("Auth header is not split by username,pw", zap.Any("auth:", splitToken[0]))
		return false
	}

	if splitToken[0] != "alcides" {
		s.logger.Error("Incorrect username")
		return false
	}

	if splitToken[1] != s.secretToken {
		s.logger.Error("Incorrect token")
		return false
	}

	return true
}
