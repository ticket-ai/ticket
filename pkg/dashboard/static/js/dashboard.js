// Guardian Dashboard JavaScript
document.addEventListener('DOMContentLoaded', function() {
    // Initialize charts with Chart.js
    initializeCharts();
    
    // Setup websocket connection for real-time updates
    setupWebSocket();
    
    // Initialize metrics refresh
    refreshMetrics();
    
    // Update last refresh time display
    updateRefreshTime();
});

// Chart colors
const chartColors = {
    primary: '#5a4fcf',
    secondary: '#3c8dbc',
    success: '#4caf50',
    danger: '#f44336',
    warning: '#ff9800',
    grid: 'rgba(255, 255, 255, 0.1)',
    text: '#a0a0a0'
};

// Global chart configurations
Chart.defaults.color = chartColors.text;
Chart.defaults.borderColor = chartColors.grid;
Chart.defaults.scale.grid.color = chartColors.grid;
Chart.defaults.scale.ticks.color = chartColors.text;

// Charts instances
let charts = {};

// Initialize all charts
function initializeCharts() {
    // Messages Per Second Chart
    const mpsCtx = document.getElementById('mps-chart').getContext('2d');
    charts.mps = new Chart(mpsCtx, {
        type: 'line',
        data: {
            labels: generateTimeLabels(30),
            datasets: [{
                label: 'Messages/sec',
                data: generateEmptyData(30),
                borderColor: chartColors.primary,
                backgroundColor: hexToRgba(chartColors.primary, 0.2),
                borderWidth: 2,
                fill: true,
                tension: 0.4,
                pointRadius: 0
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    display: false
                },
                tooltip: {
                    mode: 'index',
                    intersect: false
                }
            },
            scales: {
                x: {
                    display: true,
                    grid: {
                        display: false
                    }
                },
                y: {
                    display: true,
                    beginAtZero: true,
                    suggestedMax: 10
                }
            }
        }
    });
    
    // Other charts can be initialized here
    if (document.getElementById('latency-chart')) {
        const latencyCtx = document.getElementById('latency-chart').getContext('2d');
        charts.latency = new Chart(latencyCtx, {
            type: 'line',
            data: {
                labels: generateTimeLabels(30),
                datasets: [{
                    label: 'Avg Latency (ms)',
                    data: generateEmptyData(30),
                    borderColor: chartColors.secondary,
                    backgroundColor: hexToRgba(chartColors.secondary, 0.2),
                    borderWidth: 2,
                    fill: true,
                    tension: 0.4,
                    pointRadius: 0
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: false
                    },
                    tooltip: {
                        mode: 'index',
                        intersect: false
                    }
                },
                scales: {
                    x: {
                        display: true,
                        grid: {
                            display: false
                        }
                    },
                    y: {
                        display: true,
                        beginAtZero: true
                    }
                }
            }
        });
    }
    
    if (document.getElementById('blocked-chart')) {
        const blockedCtx = document.getElementById('blocked-chart').getContext('2d');
        charts.blocked = new Chart(blockedCtx, {
            type: 'bar',
            data: {
                labels: generateTimeLabels(30),
                datasets: [{
                    label: 'Blocked Requests',
                    data: generateEmptyData(30),
                    backgroundColor: hexToRgba(chartColors.danger, 0.6),
                    borderColor: chartColors.danger,
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: false
                    }
                },
                scales: {
                    x: {
                        display: true,
                        grid: {
                            display: false
                        }
                    },
                    y: {
                        display: true,
                        beginAtZero: true,
                        ticks: {
                            stepSize: 1
                        }
                    }
                }
            }
        });
    }
}

// Generate array of last N minutes as labels
function generateTimeLabels(count) {
    const labels = [];
    for (let i = count - 1; i >= 0; i--) {
        const time = new Date();
        time.setMinutes(time.getMinutes() - i);
        labels.push(time.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'}));
    }
    return labels;
}

// Generate empty data array
function generateEmptyData(count) {
    return Array(count).fill(0);
}

// Convert hex to rgba for transparency
function hexToRgba(hex, alpha) {
    const r = parseInt(hex.slice(1, 3), 16);
    const g = parseInt(hex.slice(3, 5), 16);
    const b = parseInt(hex.slice(5, 7), 16);
    return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

// Setup WebSocket connection
function setupWebSocket() {
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${wsProtocol}//${window.location.host}/ws`;
    
    const socket = new WebSocket(wsUrl);
    
    socket.onopen = function() {
        console.log('WebSocket connection established');
    };
    
    socket.onmessage = function(event) {
        try {
            const data = JSON.parse(event.data);
            updateDashboard(data);
        } catch (e) {
            console.error('Failed to parse WebSocket message:', e);
        }
    };
    
    socket.onclose = function() {
        console.log('WebSocket connection closed');
        // Try to reconnect after 5 seconds
        setTimeout(setupWebSocket, 5000);
    };
    
    socket.onerror = function(error) {
        console.error('WebSocket error:', error);
        socket.close();
    };
}

// Update dashboard with new data
function updateDashboard(data) {
    // Update message count metric
    if (data.metrics) {
        if (data.metrics.messagesPerSecond !== undefined) {
            document.getElementById('mps-value').textContent = data.metrics.messagesPerSecond.toFixed(2);
            
            // Update MPS chart
            const mpsChart = charts.mps;
            mpsChart.data.datasets[0].data.shift();
            mpsChart.data.datasets[0].data.push(data.metrics.messagesPerSecond);
            mpsChart.data.labels.shift();
            mpsChart.data.labels.push(new Date().toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'}));
            mpsChart.update();
        }
        
        // Update other metrics if they exist
        if (data.metrics.avgLatency !== undefined && charts.latency) {
            document.getElementById('latency-value').textContent = data.metrics.avgLatency.toFixed(2);
            
            const latencyChart = charts.latency;
            latencyChart.data.datasets[0].data.shift();
            latencyChart.data.datasets[0].data.push(data.metrics.avgLatency);
            latencyChart.data.labels.shift();
            latencyChart.data.labels.push(new Date().toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'}));
            latencyChart.update();
        }
        
        if (data.metrics.blockedRequests !== undefined && charts.blocked) {
            document.getElementById('blocked-value').textContent = data.metrics.blockedRequests;
            
            const blockedChart = charts.blocked;
            blockedChart.data.datasets[0].data.shift();
            blockedChart.data.datasets[0].data.push(data.metrics.blockedRequests);
            blockedChart.data.labels.shift();
            blockedChart.data.labels.push(new Date().toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'}));
            blockedChart.update();
        }
    }
    
    // Update activity feed
    if (data.activity && data.activity.length > 0) {
        const activityList = document.getElementById('activity-list');
        const noActivity = document.getElementById('no-activity');
        
        if (noActivity) {
            noActivity.remove();
        }
        
        // Add new activity items at the top
        data.activity.forEach(item => {
            const activityItem = document.createElement('div');
            activityItem.className = 'activity-item';
            
            const time = document.createElement('div');
            time.className = 'time';
            time.textContent = new Date().toLocaleTimeString([], {hour: '2-digit', minute:'2-digit', second:'2-digit'});
            
            const method = document.createElement('div');
            method.className = 'method';
            method.textContent = item.method || 'POST';
            
            const endpoint = document.createElement('div');
            endpoint.className = 'endpoint';
            endpoint.textContent = item.endpoint || '/v1/completions';
            
            const latency = document.createElement('div');
            latency.className = 'latency';
            latency.textContent = `${item.latency || 0}ms`;
            
            const status = document.createElement('div');
            status.className = `status ${item.status || 'ok'}`;
            status.textContent = item.status || 'OK';
            
            activityItem.appendChild(time);
            activityItem.appendChild(method);
            activityItem.appendChild(endpoint);
            activityItem.appendChild(latency);
            activityItem.appendChild(status);
            
            activityList.prepend(activityItem);
            
            // Remove oldest items if there are more than 50
            if (activityList.children.length > 50) {
                activityList.removeChild(activityList.lastChild);
            }
        });
    }
    
    // Update last refresh time
    updateRefreshTime();
}

// Refresh metrics via AJAX if WebSocket is not available
function refreshMetrics() {
    fetch('/metrics')
        .then(response => {
            if (!response.ok) {
                throw new Error('Network response was not ok');
            }
            return response.json();
        })
        .then(data => {
            updateDashboard(data);
        })
        .catch(error => {
            console.error('Error fetching metrics:', error);
        })
        .finally(() => {
            // Schedule next refresh after 5 seconds
            setTimeout(refreshMetrics, 5000);
        });
}

// Update last refresh time
function updateRefreshTime() {
    const refreshElement = document.getElementById('last-refresh');
    if (refreshElement) {
        const now = new Date();
        refreshElement.textContent = now.toLocaleTimeString();
    }
}