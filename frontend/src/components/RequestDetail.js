import React from 'react';
import { useParams } from 'react-router-dom';
import { Box, Typography, Paper, Card, CardContent } from '@mui/material';

function RequestDetail() {
  const { requestId } = useParams();
  
  return (
    <Box>
      <Typography variant="h4" gutterBottom>Request Details</Typography>
      <Card>
        <CardContent>
          <Typography variant="h6">Request ID: {requestId}</Typography>
          <Typography variant="body1">
            Detailed information about this request will appear here.
          </Typography>
        </CardContent>
      </Card>
    </Box>
  );
}

export default RequestDetail;