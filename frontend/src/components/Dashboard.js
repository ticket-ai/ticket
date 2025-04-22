import React, { useState, useEffect } from 'react';
import { Box, Typography, Paper, Grid, CircularProgress } from '@mui/material';
import { 
  LineChart, Line, XAxis, YAxis, CartesianGrid, 
  Tooltip, Legend, ResponsiveContainer, BarChart, Bar 
} from 'recharts';
import axios from 'axios';

function Dashboard() {
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState({});
  const [metrics, setMetrics] = useState({});
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        
        // Fetch stats data
        const statsResponse = await axios.get('/_guardian/api/stats');
        setStats(statsResponse.data);
        
        // Fetch metrics data
        const metricsResponse = await axios.get('/_guardian/api/metrics');
        setMetrics(metricsResponse.data);
        
        setError(null);
      } catch (err) {
        console.error("Error fetching data:", err);
        setError("Failed to load dashboard data. Please check if the API is running.");
      } finally {
        setLoading(false);
      }
    };

    // Initial fetch
    fetchData();
    
    // Set up polling every 10 seconds
    const interval = setInterval(fetchData, 10000);
    
    // Clean up on unmount
    return () => clearInterval(interval);
  }, []);

  // Transform toxicity data for chart
  const prepareNLPData = () => {
    if (!stats.toxicityScores) return [];
    
    return [
      { name: 'Toxicity', average: average(stats.toxicityScores), max: Math.max(...stats.toxicityScores) },
      { name: 'Profanity', average: average(stats.profanityScores), max: Math.max(...stats.profanityScores) },
      { name: 'PII', average: average(stats.piiScores), max: Math.max(...stats.piiScores) },
      { name: 'Bias', average: average(stats.biasScores), max: Math.max(...stats.biasScores) },
    ];
  };

  // Helper function to calculate average
  const average = arr => arr.reduce((a, b) => a + b, 0) / arr.length;

  // Transform model usage data for chart
  const prepareModelData = () => {
    if (!stats.requestsPerModel) return [];
    
    return Object.entries(stats.requestsPerModel).map(([model, count]) => ({
      name: model,
      requests: count
    }));
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="200px">
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box>
        <Typography variant="h4" gutterBottom>Guardian Dashboard</Typography>
        <Paper sx={{ p: 3, bgcolor: '#ff000015' }}>
          <Typography color="error">{error}</Typography>
        </Paper>
      </Box>
    );
  }

  return (
    <Box>
      <Typography variant="h4" gutterBottom>Guardian Dashboard</Typography>
      
      <Grid container spacing={3}>
        {/* Summary stats */}
        <Grid item xs={12} md={6} lg={3}>
          <Paper sx={{ p: 2, textAlign: 'center' }}>
            <Typography variant="h6">Total Requests</Typography>
            <Typography variant="h3">{stats.totalRequests || 0}</Typography>
          </Paper>
        </Grid>
        
        <Grid item xs={12} md={6} lg={3}>
          <Paper sx={{ p: 2, textAlign: 'center' }}>
            <Typography variant="h6">Flagged Requests</Typography>
            <Typography variant="h3" color="warning.main">{stats.flaggedRequests || 0}</Typography>
          </Paper>
        </Grid>
        
        <Grid item xs={12} md={6} lg={3}>
          <Paper sx={{ p: 2, textAlign: 'center' }}>
            <Typography variant="h6">Blocked Requests</Typography>
            <Typography variant="h3" color="error.main">{stats.blockedRequests || 0}</Typography>
          </Paper>
        </Grid>
        
        <Grid item xs={12} md={6} lg={3}>
          <Paper sx={{ p: 2, textAlign: 'center' }}>
            <Typography variant="h6">Estimated Cost</Typography>
            <Typography variant="h3">${stats.estimatedCost?.toFixed(2) || '0.00'}</Typography>
          </Paper>
        </Grid>
        
        {/* NLP metrics */}
        <Grid item xs={12} md={6}>
          <Paper sx={{ p: 2 }}>
            <Typography variant="h6" gutterBottom>NLP Analysis</Typography>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={prepareNLPData()}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" />
                <YAxis domain={[0, 1]} />
                <Tooltip />
                <Legend />
                <Bar dataKey="average" fill="#3f51b5" name="Average Score" />
                <Bar dataKey="max" fill="#f44336" name="Max Score" />
              </BarChart>
            </ResponsiveContainer>
          </Paper>
        </Grid>
        
        {/* Model usage */}
        <Grid item xs={12} md={6}>
          <Paper sx={{ p: 2 }}>
            <Typography variant="h6" gutterBottom>Model Usage</Typography>
            <ResponsiveContainer width="100%" height={300}>
              <BarChart data={prepareModelData()}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="name" />
                <YAxis />
                <Tooltip />
                <Legend />
                <Bar dataKey="requests" fill="#4caf50" name="Requests" />
              </BarChart>
            </ResponsiveContainer>
          </Paper>
        </Grid>
        
        {/* Real-time metrics */}
        <Grid item xs={12}>
          <Paper sx={{ p: 2 }}>
            <Typography variant="h6" gutterBottom>System Metrics</Typography>
            <Grid container spacing={2}>
              <Grid item xs={6} md={3}>
                <Paper sx={{ p: 1, bgcolor: 'background.default', textAlign: 'center' }}>
                  <Typography variant="body2">Requests/Second</Typography>
                  <Typography variant="h5">{metrics.requestsPerSecond?.toFixed(1) || "0.0"}</Typography>
                </Paper>
              </Grid>
              <Grid item xs={6} md={3}>
                <Paper sx={{ p: 1, bgcolor: 'background.default', textAlign: 'center' }}>
                  <Typography variant="body2">Latency (ms)</Typography>
                  <Typography variant="h5">{metrics.latencyMs || "0"}</Typography>
                </Paper>
              </Grid>
              <Grid item xs={6} md={3}>
                <Paper sx={{ p: 1, bgcolor: 'background.default', textAlign: 'center' }}>
                  <Typography variant="body2">CPU Usage (%)</Typography>
                  <Typography variant="h5">{metrics.cpuUsage?.toFixed(1) || "0.0"}</Typography>
                </Paper>
              </Grid>
              <Grid item xs={6} md={3}>
                <Paper sx={{ p: 1, bgcolor: 'background.default', textAlign: 'center' }}>
                  <Typography variant="body2">Memory (MB)</Typography>
                  <Typography variant="h5">{metrics.memoryUsageMB || "0"}</Typography>
                </Paper>
              </Grid>
            </Grid>
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
}

export default Dashboard;