import React, { useState, useEffect } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { 
  Box, Typography, Paper, Table, TableBody, TableCell, TableContainer, 
  TableHead, TableRow, Chip, Button, CircularProgress, TextField, InputAdornment 
} from '@mui/material';
import { Search as SearchIcon } from '@mui/icons-material';
import axios from 'axios';

function FlaggedRequests() {
  const [loading, setLoading] = useState(true);
  const [flaggedRequests, setFlaggedRequests] = useState([]);
  const [filter, setFilter] = useState('');
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchFlaggedRequests = async () => {
      try {
        setLoading(true);
        const response = await axios.get('/_guardian/api/flagged');
        setFlaggedRequests(response.data);
        setError(null);
      } catch (err) {
        console.error("Error fetching flagged requests:", err);
        setError("Failed to load flagged requests. Please check if the API is running.");
      } finally {
        setLoading(false);
      }
    };

    // Initial fetch
    fetchFlaggedRequests();
    
    // Set up polling every 10 seconds
    const interval = setInterval(fetchFlaggedRequests, 10000);
    
    // Clean up on unmount
    return () => clearInterval(interval);
  }, []);

  // Filter the requests based on search term
  const filteredRequests = flaggedRequests.filter(req => {
    const searchTerm = filter.toLowerCase();
    return (
      (req.ip && req.ip.toLowerCase().includes(searchTerm)) ||
      (req.endpoint && req.endpoint.toLowerCase().includes(searchTerm)) ||
      (req.matchedRules && req.matchedRules.some(rule => rule.toLowerCase().includes(searchTerm)))
    );
  });

  // Determine color based on score
  const getScoreColor = (score) => {
    if (score >= 0.8) return "error";
    if (score >= 0.5) return "warning";
    return "default";
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
        <Typography variant="h4" gutterBottom>Flagged Requests</Typography>
        <Paper sx={{ p: 3, bgcolor: '#ff000015' }}>
          <Typography color="error">{error}</Typography>
        </Paper>
      </Box>
    );
  }

  return (
    <Box>
      <Typography variant="h4" gutterBottom>Flagged Requests</Typography>
      
      <Box mb={3}>
        <TextField
          fullWidth
          variant="outlined"
          label="Filter Requests"
          placeholder="Search by IP, endpoint, or rule..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            ),
          }}
        />
      </Box>
      
      <Paper>
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Time</TableCell>
                <TableCell>IP Address</TableCell>
                <TableCell>Endpoint</TableCell>
                <TableCell>Score</TableCell>
                <TableCell>Matched Rules</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredRequests.length > 0 ? (
                filteredRequests.map((req, index) => (
                  <TableRow key={index}>
                    <TableCell>{new Date(req.timestamp).toLocaleString()}</TableCell>
                    <TableCell>{req.ip}</TableCell>
                    <TableCell>{req.endpoint}</TableCell>
                    <TableCell>
                      <Chip 
                        label={req.score.toFixed(2)} 
                        color={getScoreColor(req.score)}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      {req.matchedRules?.map((rule, i) => (
                        <Chip key={i} label={rule} size="small" sx={{ m: 0.5 }} />
                      ))}
                    </TableCell>
                    <TableCell>
                      <Button 
                        variant="contained" 
                        size="small" 
                        component={RouterLink}
                        to={`/request/${index}`}
                        state={{ request: req }}
                      >
                        Details
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={6} align="center">No flagged requests found</TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>
    </Box>
  );
}

export default FlaggedRequests;