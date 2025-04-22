import React from 'react';
import { Link as RouterLink, useLocation } from 'react-router-dom';
import { 
  AppBar, Toolbar, Typography, Button, Box, 
  Drawer, List, ListItem, ListItemIcon, ListItemText
} from '@mui/material';
import { 
  Dashboard as DashboardIcon, 
  Flag as FlagIcon,
  Settings as SettingsIcon 
} from '@mui/icons-material';

const drawerWidth = 240;

function Navigation() {
  const location = useLocation();
  
  const menuItems = [
    { text: 'Dashboard', icon: <DashboardIcon />, path: '/' },
    { text: 'Flagged Requests', icon: <FlagIcon />, path: '/flagged' },
    { text: 'Settings', icon: <SettingsIcon />, path: '/settings' },
  ];

  return (
    <>
      <AppBar position="fixed" sx={{ zIndex: (theme) => theme.zIndex.drawer + 1 }}>
        <Toolbar>
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            Guardian
          </Typography>
          <Button color="inherit" component="a" href="http://localhost:3000/d/guardian/guardian" target="_blank">
            Grafana
          </Button>
        </Toolbar>
      </AppBar>
      <Drawer
        variant="permanent"
        sx={{
          width: drawerWidth,
          flexShrink: 0,
          [`& .MuiDrawer-paper`]: { 
            width: drawerWidth, 
            boxSizing: 'border-box',
          },
        }}
      >
        <Toolbar /> {/* Empty toolbar to push content below app bar */}
        <Box sx={{ overflow: 'auto' }}>
          <List>
            {menuItems.map((item) => (
              <ListItem 
                button 
                key={item.text} 
                component={RouterLink} 
                to={item.path}
                selected={location.pathname === item.path}
              >
                <ListItemIcon>{item.icon}</ListItemIcon>
                <ListItemText primary={item.text} />
              </ListItem>
            ))}
          </List>
        </Box>
      </Drawer>
    </>
  );
}

export default Navigation;