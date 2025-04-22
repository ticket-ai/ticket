import React from 'react';
import { Box, Typography, Paper, Table, TableBody, TableCell, TableContainer, TableHead, TableRow } from '@mui/material';

function FlaggedRequests() {
  return (
    <Box>
      <Typography variant="h4" gutterBottom>Flagged Requests</Typography>
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
              </TableRow>
            </TableHead>
            <TableBody>
              <TableRow>
                <TableCell colSpan={5} align="center">No flagged requests yet</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>
    </Box>
  );
}

export default FlaggedRequests;