import React, { Component } from 'react'
import './index.css'
import '@fontsource/roboto/300.css'
import '@fontsource/roboto/400.css'
import '@fontsource/roboto/500.css'
import '@fontsource/roboto/700.css'
import { ThemeProvider } from '@mui/material/styles'
import CssBaseline from '@mui/material/CssBaseline'
import gmgTheme from '../../theme/gmgTheme'
import Home from '../Home'
import Navigation from '../Navi'

export default class App extends Component {
  constructor(props) {
    super(props)
    this.state = {
      selectedNavigationIndex: 0
    }
  }

  selectedNavigationIndexChanged = (index) => {
    this.setState({ selectedNavigationIndex: index })
  }

  renderCards() {
    switch (this.state.selectedNavigationIndex) {
      case 1: {
        return null
      }
      case 2: {
        return null
      }
      default: {
        return <Home />
      }
    }
  }

  render() {
    return (
      <ThemeProvider theme={gmgTheme}>
        <CssBaseline />
        <div className="app">
          {this.renderCards()}
          <Navigation
            onSelectedIndexChanged={this.selectedNavigationIndexChanged}
            selectedIndex={this.selectedNavigationIndex}
          />
        </div>
      </ThemeProvider>
    )
  }
}
