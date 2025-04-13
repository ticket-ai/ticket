// Package dashboard provides real-time monitoring dashboard for Guardian.
package dashboard

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/gorilla/websocket"
)

//go:embed templates/* static/*
var dashboardContent embed.FS

// Config contains configuration options for the dashboard
type Config struct {
	Port            int
	EnableWebsocket bool
	RefreshInterval time.Duration
	ServiceName     string
	Environment     string
}

// Dashboard implements a real-time monitoring dashboard for Guardian
type Dashboard struct {
	server         *http.Server
	config         Config
	metrics        *Metrics
	activityFeed   *ActivityFeed
	clients        map[*websocket.Conn]bool
	clientsMutex   sync.Mutex
	messageChannel chan []byte
}

// Metrics holds the current metrics data
type Metrics struct {
	MessagesPerSecond float64   `json:"messagesPerSecond"`
	AvgLatency        float64   `json:"avgLatency"`
	BlockedRequests   int       `json:"blockedRequests"`
	TopEndpoints      []string  `json:"topEndpoints"`
	RequestCounts     []int     `json:"requestCounts"`
	LastUpdated       time.Time `json:"lastUpdated"`
	mutex             sync.Mutex
}

// ActivityItem represents a single activity entry
type ActivityItem struct {
	Time      time.Time `json:"time"`
	Method    string    `json:"method"`
	Endpoint  string    `json:"endpoint"`
	Latency   int       `json:"latency"`
	Status    string    `json:"status"`
	IPAddress string    `json:"ipAddress"`
}

// ActivityFeed holds the recent activity data
type ActivityFeed struct {
	Items []ActivityItem `json:"items"`
	mutex sync.Mutex
}

// WebsocketMessage is the structure of messages sent to clients
type WebsocketMessage struct {
	Metrics  *Metrics       `json:"metrics,omitempty"`
	Activity []ActivityItem `json:"activity,omitempty"`
}

// New creates a new Dashboard
func New(config Config) *Dashboard {
	// Set default values
	if config.Port == 0 {
		config.Port = 8888
	}
	if config.RefreshInterval == 0 {
		config.RefreshInterval = 5 * time.Second
	}

	return &Dashboard{
		config: config,
		metrics: &Metrics{
			MessagesPerSecond: 0,
			AvgLatency:        0,
			BlockedRequests:   0,
			TopEndpoints:      []string{},
			RequestCounts:     []int{},
			LastUpdated:       time.Now(),
		},
		activityFeed:   &ActivityFeed{Items: []ActivityItem{}},
		clients:        make(map[*websocket.Conn]bool),
		messageChannel: make(chan []byte, 256),
	}
}

// Start initializes and starts the dashboard server
func (d *Dashboard) Start() string {
	// Setup routes
	mux := http.NewServeMux()

	// Register static file handlers
	mux.HandleFunc("/static/", d.staticHandler)

	// Register API routes
	mux.HandleFunc("/api/metrics", d.metricsHandler)
	mux.HandleFunc("/api/activity", d.activityHandler)

	// Register WebSocket endpoint
	if d.config.EnableWebsocket {
		mux.HandleFunc("/ws", d.websocketHandler)
	}

	// Register main dashboard route
	mux.HandleFunc("/", d.dashboardHandler)

	// Create server
	d.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", d.config.Port),
		Handler: mux,
	}

	// Start WebSocket broadcaster
	if d.config.EnableWebsocket {
		go d.broadcastUpdates()
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Guardian Dashboard is running at http://localhost:%d", d.config.Port)
		if err := d.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Dashboard server error: %v", err)
		}
	}()

	// Return URL for accessing the dashboard
	return fmt.Sprintf("http://localhost:%d", d.config.Port)
}

// Stop shuts down the dashboard server
func (d *Dashboard) Stop() error {
	// Close all WebSocket connections
	d.clientsMutex.Lock()
	for client := range d.clients {
		client.Close()
	}
	d.clientsMutex.Unlock()

	// Shutdown the server
	if d.server != nil {
		return d.server.Close()
	}
	return nil
}

// UpdateMetrics updates the current metrics
func (d *Dashboard) UpdateMetrics(messagesPerSecond float64, avgLatency float64, blockedRequests int, topEndpoints []string, requestCounts []int) {
	d.metrics.mutex.Lock()
	defer d.metrics.mutex.Unlock()

	d.metrics.MessagesPerSecond = messagesPerSecond
	d.metrics.AvgLatency = avgLatency
	d.metrics.BlockedRequests = blockedRequests
	d.metrics.TopEndpoints = topEndpoints
	d.metrics.RequestCounts = requestCounts
	d.metrics.LastUpdated = time.Now()

	// Broadcast updates if WebSocket is enabled
	if d.config.EnableWebsocket {
		d.broadcastMetrics()
	}
}

// AddActivity adds a new activity item to the feed
func (d *Dashboard) AddActivity(method, endpoint string, latency int, status, ipAddress string) {
	d.activityFeed.mutex.Lock()
	defer d.activityFeed.mutex.Unlock()

	// Add new item
	item := ActivityItem{
		Time:      time.Now(),
		Method:    method,
		Endpoint:  endpoint,
		Latency:   latency,
		Status:    status,
		IPAddress: ipAddress,
	}

	// Add to beginning of slice
	d.activityFeed.Items = append([]ActivityItem{item}, d.activityFeed.Items...)

	// Limit to last 100 items
	if len(d.activityFeed.Items) > 100 {
		d.activityFeed.Items = d.activityFeed.Items[:100]
	}

	// Broadcast updates if WebSocket is enabled
	if d.config.EnableWebsocket {
		d.broadcastActivity([]ActivityItem{item})
	}
}

// GetDashboardURL returns the URL where the dashboard can be accessed
func (d *Dashboard) GetDashboardURL() string {
	return fmt.Sprintf("http://localhost:%d", d.config.Port)
}

// RecordRequest adds a new request to the activity feed
func (d *Dashboard) RecordRequest(method, endpoint string, latency int, status string, ipAddress string) {
	d.activityFeed.mutex.Lock()
	defer d.activityFeed.mutex.Unlock()

	// Create a new activity item
	item := ActivityItem{
		Time:      time.Now(),
		Method:    method,
		Endpoint:  endpoint,
		Latency:   latency,
		Status:    status,
		IPAddress: ipAddress,
	}

	// Add to the beginning of the list
	d.activityFeed.Items = append([]ActivityItem{item}, d.activityFeed.Items...)

	// Trim the list if it gets too long
	if len(d.activityFeed.Items) > 100 {
		d.activityFeed.Items = d.activityFeed.Items[:100]
	}

	// Update metrics as needed
	d.metrics.mutex.Lock()
	d.metrics.LastUpdated = time.Now()
	d.metrics.mutex.Unlock()
}

// IncrementMessagesPerSecond increases the messages per second metric
func (d *Dashboard) IncrementMessagesPerSecond(value float64) {
	d.metrics.mutex.Lock()
	defer d.metrics.mutex.Unlock()

	// Simple incrementing for now, could be made more sophisticated
	d.metrics.MessagesPerSecond += value
	d.metrics.LastUpdated = time.Now()
}

// updateClients sends the current state to all websocket clients
func (d *Dashboard) updateClients() {
	message := WebsocketMessage{
		Metrics:  d.metrics,
		Activity: d.activityFeed.Items,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling websocket message: %v", err)
		return
	}

	select {
	case d.messageChannel <- data:
		// Message sent to channel
	default:
		// Channel buffer is full, skip this update
		log.Println("Websocket message channel is full, skipping update")
	}
}

// dashboardHandler serves the main dashboard HTML
func (d *Dashboard) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Load dashboard template
	tmplPath := "templates/dashboard.html"
	log.Printf("Looking for dashboard template at: %s", tmplPath)
	
	content, err := dashboardContent.ReadFile(tmplPath)
	if err != nil {
		log.Printf("Error reading dashboard template: %v", err)
		http.Error(w, "Dashboard template not found", http.StatusInternalServerError)
		return
	}
	
	log.Printf("Dashboard template found, size: %d bytes", len(content))

	// Create template from content
	tmpl, err := template.New("dashboard").Parse(string(content))
	if err != nil {
		log.Printf("Error parsing dashboard template: %v", err)
		http.Error(w, "Error parsing dashboard template", http.StatusInternalServerError)
		return
	}

	// Prepare template data
	data := map[string]interface{}{
		"ServiceName":      d.config.ServiceName,
		"Environment":      d.config.Environment,
		"WebSocketEnabled": d.config.EnableWebsocket,
		"RefreshInterval":  int(d.config.RefreshInterval.Seconds()),
		"Title":            "Guardian Dashboard",
		"Metrics":          d.metrics,
		"Activity":         d.activityFeed.Items,
	}

	// Execute template with data
	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing dashboard template: %v", err)
		http.Error(w, "Error rendering dashboard", http.StatusInternalServerError)
		return
	}
}

// staticHandler serves static files (CSS, JS)
func (d *Dashboard) staticHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the file path from the URL
	filePath := r.URL.Path[len("/static/"):]
	fullPath := filepath.Join("static", filePath)

	// Check if file exists
	content, err := dashboardContent.ReadFile(fullPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Determine content type
	var contentType string
	if strings.HasSuffix(fullPath, ".css") {
		contentType = "text/css"
	} else if strings.HasSuffix(fullPath, ".js") {
		contentType = "application/javascript"
	} else if strings.HasSuffix(fullPath, ".png") {
		contentType = "image/png"
	} else if strings.HasSuffix(fullPath, ".jpg") || strings.HasSuffix(fullPath, ".jpeg") {
		contentType = "image/jpeg"
	} else {
		contentType = "text/plain"
	}

	// Serve the file
	w.Header().Set("Content-Type", contentType)
	w.Write(content)
}

// metricsHandler handles API requests for metrics data
func (d *Dashboard) metricsHandler(w http.ResponseWriter, r *http.Request) {
	d.metrics.mutex.Lock()
	data := d.metrics
	d.metrics.mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// activityHandler handles API requests for activity data
func (d *Dashboard) activityHandler(w http.ResponseWriter, r *http.Request) {
	d.activityFeed.mutex.Lock()
	data := d.activityFeed.Items
	d.activityFeed.mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// WebSocket connection upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections
	},
}

// websocketHandler handles WebSocket connections
func (d *Dashboard) websocketHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade the connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Register client
	d.clientsMutex.Lock()
	d.clients[conn] = true
	d.clientsMutex.Unlock()

	// Handle disconnection
	defer func() {
		d.clientsMutex.Lock()
		delete(d.clients, conn)
		d.clientsMutex.Unlock()
		conn.Close()
	}()

	// Send initial data
	d.sendInitialData(conn)

	// Keep the connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// sendInitialData sends current metrics and activity to a new client
func (d *Dashboard) sendInitialData(conn *websocket.Conn) {
	// Prepare message with current data
	d.metrics.mutex.Lock()
	d.activityFeed.mutex.Lock()

	message := WebsocketMessage{
		Metrics:  d.metrics,
		Activity: d.activityFeed.Items,
	}

	d.metrics.mutex.Unlock()
	d.activityFeed.mutex.Unlock()

	// Marshal to JSON
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling initial data: %v", err)
		return
	}

	// Send to client
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("Error sending initial data: %v", err)
	}
}

// broadcastMetrics sends current metrics to all connected clients
func (d *Dashboard) broadcastMetrics() {
	d.metrics.mutex.Lock()
	message := WebsocketMessage{
		Metrics: d.metrics,
	}
	d.metrics.mutex.Unlock()

	// Marshal to JSON
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling metrics: %v", err)
		return
	}

	// Send to message channel
	d.messageChannel <- data
}

// broadcastActivity sends new activity items to all connected clients
func (d *Dashboard) broadcastActivity(items []ActivityItem) {
	message := WebsocketMessage{
		Activity: items,
	}

	// Marshal to JSON
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling activity: %v", err)
		return
	}

	// Send to message channel
	d.messageChannel <- data
}

// broadcastUpdates processes messages from the channel and sends them to clients
func (d *Dashboard) broadcastUpdates() {
	for message := range d.messageChannel {
		d.clientsMutex.Lock()
		for client := range d.clients {
			if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Error broadcasting to client: %v", err)
				client.Close()
				delete(d.clients, client)
			}
		}
		d.clientsMutex.Unlock()
	}
}

// URL returns the current dashboard URL
func (d *Dashboard) URL() string {
	return fmt.Sprintf("http://localhost:%d", d.config.Port)
}
