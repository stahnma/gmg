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

export default class GrillTemperature extends Component {
  constructor(props) {
    super(props)
    this.state = {
      open: false,
      desiredGrillTemp: '',
      desiredGrillTempError: ''
    }
  }

  handleOpen = () => {
    if (this.props.isEnabled) {
      this.setState({ open: true })
    }
  }

  handleCancel = () => this.setState({ open: false, desiredGrillTemp: 0 })

  handleSubmit = () => {
    this.setState({ open: false })
    this.props.onSubmit(this.state.desiredGrillTemp)
  }

  handleDesiredGrillTempChange = (event) => {
    const value = event.target.value
    let error = ''
    if (isNaN(value)) error = 'Desired temperature must be a number!'
    else if (value < 0) error = 'Desired temperature must be greater than 0 ℉!'
    else if (value > 500) error = 'Desired temperature must be less than 500 ℉!'
    this.setState({
      desiredGrillTemp: value,
      desiredGrillTempError: error
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
            <Typography variant="h6">Grill Temp ℉</Typography>
            <Typography variant="body2">Set the temperature of the grill</Typography>
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
              <ListItemText primary={`Current: ${this.props.currentGrillTemp} ℉`} />
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
                primary={`Desired: ${this.props.desiredGrillTemp
                  ? `${this.props.desiredGrillTemp} ℉`
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
              Set Grill Temperature
            </Button>
          </CardActions>
          <Dialog open={this.state.open} onClose={this.handleSubmit}>
            <DialogTitle>Set the desired grill temperature</DialogTitle>
            <DialogContent>
              <TextField
                autoFocus
                margin="dense"
                id="desired-grill-temp"
                label="Example: 225"
                placeholder="Grill temperature ℉"
                value={this.state.desiredGrillTemp || ''}
                onChange={this.handleDesiredGrillTempChange}
                error={!!this.state.desiredGrillTempError}
                helperText={this.state.desiredGrillTempError}
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

GrillTemperature.propTypes = {
  currentGrillTemp: PropTypes.number,
  desiredGrillTemp: PropTypes.number,
  isEnabled: PropTypes.bool,
  onSubmit: PropTypes.func
}
