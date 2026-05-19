import React, { Component } from 'react'
import PropTypes from 'prop-types'
import BottomNavigation from '@mui/material/BottomNavigation'
import BottomNavigationAction from '@mui/material/BottomNavigationAction'
import Paper from '@mui/material/Paper'
import HomeIcon from '@mui/icons-material/Home'
import './index.css'

export default class Navigation extends Component {
  constructor(props) {
    super(props)
    this.state = {
      selectedIndex: props.selectedIndex || 0
    }
  }

  select = (index) => {
    if (this.props.onSelectedIndexChanged) {
      this.props.onSelectedIndexChanged(index)
    }
    return this.setState({ selectedIndex: index })
  }

  render() {
    return (
      <div className="navi">
        <Paper elevation={1}>
          <BottomNavigation
            showLabels
            value={this.state.selectedIndex}
            onChange={(event, index) => this.select(index)}
          >
            <BottomNavigationAction label="Home" icon={<HomeIcon />} />
            {/* <BottomNavigationAction label="Profiles" icon={<TableChartIcon />} />
            <BottomNavigationAction label="Settings" icon={<SettingsIcon />} /> */}
          </BottomNavigation>
        </Paper>
      </div>
    )
  }
}

Navigation.propTypes = {
  onSelectedIndexChanged: PropTypes.func,
  selectedIndex: PropTypes.number
}
