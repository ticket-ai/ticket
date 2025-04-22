import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import Box from '@mui/material/Box';

// Import components
import Dashboard from './components/Dashboard';
import FlaggedRequests from './components/FlaggedRequests';
import RequestDetail from './components/RequestDetail';
import Navigation from './components/Navigation';
import Settings from './components/Settings';

// Create a dark theme
const darkTheme = createTheme({
  palette: {
    mode: 'dark',
    primary: {
      main: '#90caf9',
    },
    secondary: {
      main: '#f48fb1',
    },
    background: {
      default: '#121212',
      paper: '#1e1e1e',
    },
  },
  typography: {
    h4: {
      fontWeight: 500,
    },
    h6: {
      fontWeight: 500,
    }
  },
});

function App() {
  return (
    <ThemeProvider theme={darkTheme}>
      <CssBaseline />
      <Router>
        <Box sx={{ display: 'flex' }}>
          <Navigation />
          <Box component="main" sx={{ 
            flexGrow: 1, 
            p: 3, 
            mt: 8, // Margin top to account for AppBar height
            ml: { sm: '240px' } // Margin left to account for drawer width on non-mobile
          }}>
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/flagged" element={<FlaggedRequests />} />
              <Route path="/request/:requestId" element={<RequestDetail />} />
              <Route path="/settings" element={<Settings />} />
            </Routes>
          </Box>
        </Box>
      </Router>
    </ThemeProvider>
  );
}

export default App;