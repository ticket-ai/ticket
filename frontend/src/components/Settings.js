import React from 'react';
import { Box, Typography, Paper, List, ListItem, ListItemText, Switch } from '@mui/material';

function Settings() {
  return (
    <Box>
      <Typography variant="h4" gutterBottom>Settings</Typography>
      <Paper>
        <List>
          <ListItem>
            <ListItemText primary="Enable NLP Analysis" secondary="Perform natural language processing on requests" />
            <Switch edge="end" defaultChecked />
          </ListItem>
          <ListItem>
            <ListItemText primary="Auto Block High Risk Requests" secondary="Automatically block requests above threshold" />
            <Switch edge="end" defaultChecked />
          </ListItem>
          <ListItem>
            <ListItemText primary="Send Email Alerts" secondary="Email notifications for flagged requests" />
            <Switch edge="end" />
          </ListItem>
        </List>
      </Paper>
    </Box>
  );
}

export default Settings;