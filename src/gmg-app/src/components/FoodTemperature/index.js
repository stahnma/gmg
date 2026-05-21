import React, { Component } from 'react'
import PropTypes from 'prop-types'
import Card from '@mui/material/Card'
import CardActions from '@mui/material/CardActions'
import CardMedia from '@mui/material/CardMedia'
import Avatar from '@mui/material/Avatar'
import Box from '@mui/material/Box'
import List from '@mui/material/List'
import ListItem from '@mui/material/ListItem'
import ListItemAvatar from '@mui/material/ListItemAvatar'
import ListItemText from '@mui/material/ListItemText'
import Button from '@mui/material/Button'
import Dialog from '@mui/material/Dialog'
import DialogTitle from '@mui/material/DialogTitle'
import DialogContent from '@mui/material/DialogContent'
import DialogActions from '@mui/material/DialogActions'
import TextField from '@mui/material/TextField'
import Typography from '@mui/material/Typography'
import ThermostatIcon from '@mui/icons-material/Thermostat'
import DeviceThermostatIcon from '@mui/icons-material/DeviceThermostat'
import logo from './logo.png'
import '@fontsource/roboto/300.css'
import '@fontsource/roboto/400.css'
import '@fontsource/roboto/500.css'
import '@fontsource/roboto/700.css'
import './index.css'

export default class FoodTemperature extends Component {
  constructor(props) {
    super(props)
    this.state = {
      open: false,
      desiredFoodTemp: '',
      desiredFoodTempError: ''
    }
  }

  handleOpen = () => {
    if (this.props.isEnabled) {
      this.setState({ open: true })
    }
  }

  handleCancel = () => this.setState({ open: false, desiredFoodTemp: 0 })

  handleSubmit = () => {
    this.setState({ open: false })
    this.props.onSubmit(this.state.desiredFoodTemp)
  }

  handleDesiredTempChange = (event) => {
    const value = event.target.value
    let error = ''
    if (isNaN(value)) error = 'Desired temperature must be a number!'
    else if (value < 0) error = 'Desired temperature must be greater than 0 ℉!'
    else if (value > 500) error = 'Desired temperature must be less than 500 ℉!'
    this.setState({
      desiredFoodTemp: value,
      desiredFoodTempError: error
    })
  }

  render() {
    return (
      <Card>
        <Box sx={{ position: 'relative' }}>
          <CardMedia component="img" image={logo} alt="" />
          <Box
            sx={{
              position: 'absolute',
              bottom: 0,
              left: 0,
              right: 0,
              bgcolor: 'rgba(0, 0, 0, 0.55)',
              px: 2,
              py: 1
            }}
          >
            <Typography variant="h6">Food Temp ℉</Typography>
            <Typography variant="body2">Set the temperature of the food</Typography>
          </Box>
        </Box>
        <div className="controls">
          <List>
            <ListItem>
              <ListItemAvatar>
                <Avatar sx={{ width: 50, height: 50 }}>
                  <ThermostatIcon />
                </Avatar>
              </ListItemAvatar>
              <ListItemText primary={`Current: ${this.props.currentFoodTemp} ℉`} />
            </ListItem>
          </List>
          <List>
            <ListItem>
              <ListItemAvatar>
                <Avatar sx={{ width: 50, height: 50 }}>
                  <DeviceThermostatIcon />
                </Avatar>
              </ListItemAvatar>
              <ListItemText
                primary={`Desired: ${this.props.desiredFoodTemp
                  ? `${this.props.desiredFoodTemp} ℉`
                  : 'Not set'}`}
              />
            </ListItem>
          </List>
          <CardActions>
            <Button
              variant="contained"
              onClick={this.handleOpen}
              disabled={!this.props.isEnabled}
            >
              Set Food Temperature
            </Button>
          </CardActions>
          <Dialog open={this.state.open} onClose={this.handleSubmit}>
            <DialogTitle>Set the desired food temperature</DialogTitle>
            <DialogContent>
              <TextField
                autoFocus
                margin="dense"
                id="desired-food-temp"
                label="Example: 225"
                placeholder="Grill temperature ℉"
                value={this.state.desiredFoodTemp || ''}
                onChange={this.handleDesiredTempChange}
                error={!!this.state.desiredFoodTempError}
                helperText={this.state.desiredFoodTempError}
                fullWidth
                variant="standard"
              />
            </DialogContent>
            <DialogActions>
              <Button onClick={this.handleCancel}>Cancel</Button>
              <Button onClick={this.handleSubmit}>Set</Button>
            </DialogActions>
          </Dialog>
        </div>
      </Card>
    )
  }
}

FoodTemperature.propTypes = {
  currentFoodTemp: PropTypes.number,
  desiredFoodTemp: PropTypes.number,
  isEnabled: PropTypes.bool,
  onSubmit: PropTypes.func
}
