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
import TimerIcon from '@mui/icons-material/Timer'
import HourglassBottomIcon from '@mui/icons-material/HourglassBottom'
import logo from './logo.png'
import './index.css'
import '@fontsource/roboto/300.css'
import '@fontsource/roboto/400.css'
import '@fontsource/roboto/500.css'
import '@fontsource/roboto/700.css'

const regex = /(\d{1,2}):(\d{1,2}):(\d{1,2}):(\d{1,2})/g
const conversions = {
  SECONDS_IN_DAYS: 86400,
  SECONDS_IN_HOURS: 3600,
  SECONDS_IN_MIN: 60,
  SECONDS_IN_SECONDS: 1
}

export default class Timers extends Component {
  constructor(props) {
    super(props)
    this.state = {
      open: false,
      desiredCountDown: '00:00:00:00',
      desiredCountDownError: '',
      countDown: 0,
      countDownActive: false,
      countUp: 0,
      countUpActive: false
    }
  }

  componentWillUnmount() {
    if (this.countDownSchedule) clearInterval(this.countDownSchedule)
    if (this.countUpSchedule) clearInterval(this.countUpSchedule)
  }

  handleOpen = () => this.setState({ open: true })
  handleCancel = () => this.setState({ open: false, desiredCountDown: 0 })
  handleSubmit = () => {
    try {
      const seconds = this.computeSeconds(this.state.desiredCountDown)
      this.setState({ open: false })
      this.countDown(seconds)
    }
    catch (ex) {
      this.setState({ desiredCountDownError: 'Invalid input' })
    }

  }

  handleDesiredCountDownChange = (event) => {
    this.setState({
      desiredCountDown: event.target.value,
      desiredCountDownError: ''
    })
  }

  countDown = (seconds) => {
    this.setState({ countDownActive: true, countDown: seconds })
    this.countDownSchedule = setInterval(() => {
      if (this.state.countDownActive && this.state.countDown > 0) {
        this.setState({ countDown: this.state.countDown - 1 })
      }
      else {
        clearInterval(this.countDownSchedule)
      }
    }, 1000)
  }

  cancelCountDown = () => this.setState({ countDownActive: false, countDown: 0 })

  countUp = () => {
    this.setState({ countUpActive: true })
    this.countUpSchedule = setInterval(() => {
      if (this.state.countUpActive) {
        this.setState({ countUp: this.state.countUp + 1 })
      }
      else {
        clearInterval(this.countUpSchedule)
      }
    }, 1000)
  }

  cancelCountUp = () => this.setState({ countUpActive: false, countUp: 0 })

  computeSeconds = (input) => {
    let result = 0
    const parseResult = (matches) => {
      if (matches.index === regex.lastIndex) regex.lastIndex++
      matches.filter((match, groupIndex) => groupIndex > 0).forEach((match, groupIndex) => {
        const key = Object.keys(conversions)[groupIndex]
        const conversion = conversions[key]
        result += conversion * parseInt(match, 10)
      })
    }

    if (!input.includes(':') && !isNaN(parseInt(input, 10))) {
      result = parseInt(input, 10)
    }
    else {
      let matches = regex.exec(input)
      while (matches !== null) {
        parseResult(matches)
        matches = regex.exec(input)
      }
    }

    if (!result) throw new Error('Input could not be parsed!')
    return result
  }

  formatSeconds = (seconds) => {
    const days = Math.trunc(seconds / conversions.SECONDS_IN_DAYS)
    seconds -= conversions.SECONDS_IN_DAYS * days

    const hours = Math.trunc(seconds / conversions.SECONDS_IN_HOURS)
    seconds -= conversions.SECONDS_IN_HOURS * hours

    const mins = Math.trunc(seconds / conversions.SECONDS_IN_MIN)
    seconds -= conversions.SECONDS_IN_MIN * mins

    const pad = (num, size) => {
      var s = num + ""
      while (s.length < size) s = "0" + s
      return s
    }

    return `${pad(days, 2)}:${pad(hours, 2)}:${pad(mins, 2)}:${pad(seconds, 2)}`
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
            <Typography variant="h6">Timers</Typography>
            <Typography variant="body2">Set a grilling stopwatch or countdown timer.</Typography>
          </Box>
        </Box>
        <div className="controls">
          <List>
            <ListItem>
              <ListItemAvatar>
                <Avatar sx={{ width: 50, height: 50 }}>
                  <TimerIcon />
                </Avatar>
              </ListItemAvatar>
              <ListItemText primary={`Timer: ${this.formatSeconds(this.state.countUp)}`} />
            </ListItem>
          </List>
          <CardActions>
            <Button
              variant="contained"
              onClick={() => this.state.countUpActive ? this.cancelCountUp() : this.countUp()}
              disabled={!this.props.isEnabled}
            >
              {this.state.countUpActive ? 'Cancel' : 'Start'}
            </Button>
          </CardActions>
          <List>
            <ListItem>
              <ListItemAvatar>
                <Avatar sx={{ width: 50, height: 50 }}>
                  <HourglassBottomIcon />
                </Avatar>
              </ListItemAvatar>
              <ListItemText primary={`Countdown: ${this.formatSeconds(this.state.countDown)}`} />
            </ListItem>
          </List>
          <CardActions>
            <Button
              variant="contained"
              onClick={() => this.state.countDownActive ? this.cancelCountDown() : this.handleOpen()}
              disabled={!this.props.isEnabled}
            >
              {this.state.countDownActive ? 'Cancel' : 'Start'}
            </Button>
          </CardActions>
          <Dialog open={this.state.open} onClose={this.handleSubmit}>
            <DialogTitle>Set the countdown time (dd:hh:mm:ss)</DialogTitle>
            <DialogContent>
              <TextField
                margin="dense"
                id="desired-countdown"
                label="00:01:30:00"
                placeholder="Set countdown"
                value={this.state.desiredCountDown || ''}
                onChange={this.handleDesiredCountDownChange}
                error={!!this.state.desiredCountDownError}
                helperText={this.state.desiredCountDownError}
                fullWidth
                variant="standard"
              />
            </DialogContent>
            <DialogActions>
              <Button onClick={this.handleCancel}>Cancel</Button>
              <Button
                disabled={!!this.state.desiredCountDownError}
                onClick={this.handleSubmit}
              >
                Set
              </Button>
            </DialogActions>
          </Dialog>
        </div>
      </Card>
    )
  }
}

Timers.propTypes = {
  isEnabled: PropTypes.bool
}
