import React, { Component } from 'react'
import PropTypes from 'prop-types'
import Box from '@mui/material/Box'
import IconButton from '@mui/material/IconButton'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import LinearProgress from '@mui/material/LinearProgress'
import PowerSettingsNewIcon from '@mui/icons-material/PowerSettingsNew'
import AccessTimeIcon from '@mui/icons-material/AccessTime'
import HistoryIcon from '@mui/icons-material/History'
import WifiIcon from '@mui/icons-material/Wifi'
import WarningIcon from '@mui/icons-material/Warning'
import './index.css'
import '@fontsource/roboto/300.css'
import '@fontsource/roboto/400.css'
import '@fontsource/roboto/500.css'
import '@fontsource/roboto/700.css'

const getButtonColor = (enabled) => enabled ? '#00ff00' : 'rgb(238, 238, 238)'
const getWifiColor = (enabled) => enabled ? '#00ff00' : 'rgb(113, 113, 113)'
const getAlertColor = (enabled) => enabled ? 'rgb(255, 204, 0)' : 'rgb(113, 113, 113)'

const controlSx = { display: 'flex', flexDirection: 'column', alignItems: 'center' }
const labelSx = { color: 'rgb(238, 238, 238)', fontSize: '14px', mt: '-6px' }

export default class HomeControls extends Component {
  getAlert = () => {
    if (this.props.fanModeActive) return 'Fan mode is active!'
    if (this.props.lowPelletAlarmActive) return 'Low Pellet alarm is active!'
    return ''
  }

  render() {
    const alertText = this.getAlert()
    return (
      <div>
        <Box
          sx={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            minHeight: '5rem',
            px: 1
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box sx={controlSx}>
              <Tooltip
                title={this.props.fanModeActive ? 'Grill cannot be powered on during fan mode!' : ''}
              >
                <span>
                  <IconButton
                    disabled={!this.props.grillConnected || this.props.fanModeActive}
                    onClick={this.props.onPowerTouchTap}
                  >
                    <PowerSettingsNewIcon
                      fontSize="large"
                      sx={{ color: getButtonColor(this.props.powerOn) }}
                    />
                  </IconButton>
                </span>
              </Tooltip>
              <Typography sx={labelSx}>Power</Typography>
            </Box>
            <Box sx={controlSx}>
              <IconButton disabled={this.props.loading} onClick={this.props.onTimersTouchTap}>
                <AccessTimeIcon
                  fontSize="large"
                  sx={{ color: getButtonColor(this.props.timersOn) }}
                />
              </IconButton>
              <Typography sx={labelSx}>Timers</Typography>
            </Box>
            <Box sx={controlSx}>
              <IconButton disabled={this.props.loading} onClick={this.props.onHistoryTouchTap}>
                <HistoryIcon
                  fontSize="large"
                  sx={{ color: getButtonColor(this.props.historyOn) }}
                />
              </IconButton>
              <Typography sx={labelSx}>History</Typography>
            </Box>
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'center' }}>
            <Tooltip
              title={this.props.grillConnected
                ? 'The grill is connected!'
                : 'The grill is not connected! The application will continue trying to connect in the background.'}
            >
              <span>
                <IconButton disableRipple>
                  <WifiIcon sx={{ color: getWifiColor(this.props.grillConnected) }} />
                </IconButton>
              </span>
            </Tooltip>
            <Tooltip title={alertText}>
              <span>
                <IconButton disableRipple disabled={!alertText}>
                  <WarningIcon sx={{ color: getAlertColor(alertText) }} />
                </IconButton>
              </span>
            </Tooltip>
          </Box>
        </Box>
        {this.props.loading && <LinearProgress />}
      </div>
    )
  }
}

HomeControls.propTypes = {
  powerOn: PropTypes.bool,
  onPowerTouchTap: PropTypes.func,
  timersOn: PropTypes.bool,
  historyOn: PropTypes.bool,
  onTimersTouchTap: PropTypes.func,
  onHistoryTouchTap: PropTypes.func,
  loading: PropTypes.bool,
  fanModeActive: PropTypes.bool,
  lowPelletAlarmActive: PropTypes.bool,
  grillConnected: PropTypes.bool
}

HomeControls.defaultProps = {
  loading: true
}
