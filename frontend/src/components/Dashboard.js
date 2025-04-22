import React from 'react';
import { Box, Typography, Paper, Grid } from '@mui/material';

function Dashboard() {
  return (
    <Box>
      <Typography variant="h4" gutterBottom>Guardian Dashboard</Typography>
      <Grid container spacing={3}>
        <Grid item xs={12} md={6}>
          <Paper sx={{ p: 2 }}>
            <Typography variant="h6">Welcome to Guardian</Typography>
            <Typography variant="body1">
              This is your central dashboard for monitoring AI safety.
            </Typography>
          </Paper>
        </Grid>
        <Grid item xs={12} md={6}>
          <Paper sx={{ p: 2 }}>
            <Typography variant="h6">Quick Stats</Typography>
            <Typography variant="body1">
              Dashboard statistics will appear here once connected to the API.
            </Typography>
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
}

export default Dashboard;