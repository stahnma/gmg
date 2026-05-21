import React, { Component } from 'react'
import GrillTemperature from '../GrillTemperature'
import FoodTemperature from '../FoodTemperature'
import GrillHistory from '../GrillHistory'
import Timers from "../Timers/index"
import HomeControls from '../HomeControls'
import io from 'socket.io-client'
import GrillClient from '../../utils/GrillClient'
import { ToastContainer, toast } from 'react-toastify'
import 'react-toastify/dist/ReactToastify.css'
import Connecting from './Connecting'
import './index.css'
import '@fontsource/roboto/300.css'
import '@fontsource/roboto/400.css'
import '@fontsource/roboto/500.css'
import '@fontsource/roboto/700.css'

const GRILL_TEMPERATURE_DATASET = 0
const FOOD_TEMPERATURE_DATASET = 1
const HISTORY_WINDOW_HOURS = 8

const formatAlert = ({ name, reason }) => (
  <>
    <strong>{name}</strong>
    <br />
    {reason}
  </>
)

export default class Home extends Component {
  constructor() {
    super()
    this.client = new GrillClient(window.location.origin)
    this.state = {
      datasets: [{
        label: 'Grill Temp',
        borderColor: 'rgb(255, 99, 132)',
        backgroundColor: 'rgba(255, 99, 132, 0.5)',
        fill: false,
        pointRadius: 0,
        data: []
      }, {
        label: 'Food Temp',
        borderColor: 'rgb(54, 162, 235)',
        backgroundColor: 'rgba(54, 162, 235, 0.5)',
        fill: false,
        pointRadius: 0,
        data: []
      }],
      currentGrillTemp: 0,
      desiredGrillTemp: 0,
      currentFoodTemp: 0,
      desiredFoodTemp: 0,
      commandsPending: 0,
      lowPelletAlarmActive: false,
      fanModeActive: false,
      loading: false,
      grillConnected: false,
      socketConnected: false,
      showTimers: false,
      showHistory: false
    }

    // Setup Socket.IO
    this.socket = io(window.location.origin)
    this.socket.on('connect', () => this.setState({ socketConnected: true }))
    this.socket.on('disconnect', () => this.setState({ socketConnected: false }))
  }

  sendAlert = (alert) => {
    const body = formatAlert(alert)
    switch (alert.level) {
      case 'error':   return toast.error(body)
      case 'warning': return toast.warning(body)
      default:        return toast.info(body)
    }
  }

  componentDidMount() {
    this.socket.on('status', status => {
      this.state.datasets[GRILL_TEMPERATURE_DATASET].data.push({
        x: Date.now(),
        y: status.currentGrillTemp,
      })

      this.state.datasets[FOOD_TEMPERATURE_DATASET].data.push({
        x: Date.now(),
        y: status.currentFoodTemp,
      })

      this.setState({
        ...status,
        grillConnected: true,
        loading: !!this.state.commandsPending,
        datasets: this.state.datasets
      })
    })

    this.socket.on('alert', alert => {
      this.sendAlert(alert)
    })

    const since = Math.floor((Date.now() - HISTORY_WINDOW_HOURS * 3600 * 1000) / 1000)
    this.client.getTemperatureHistory(since).then(history => {
      if (history.length === 0) return

      this.setState({
        datasets: [{
          ...this.state.datasets[GRILL_TEMPERATURE_DATASET],
          data: history.map(d => ({ x: d.timestamp * 1000, y: d.grill_temperature }))
        }, {
          ...this.state.datasets[FOOD_TEMPERATURE_DATASET],
          data: history.map(d => ({ x: d.timestamp * 1000, y: d.food_temperature }))
        }]
      })
    })
  }

  componentWillUnmount() {
    this.socket.removeAllListeners('status')
  }

  get canChangeTemp() {
    return this.state.grillConnected &&
      this.state.isOn &&
      !this.state.loading &&
      !this.state.fanModeActive
  }

  get canPowerOn() {
    return this.state.grillConnected &&
      !this.state.loading &&
      !this.state.fanModeActive
  }

  powerToggle = async () => {
    if (!this.canPowerOn) return
    this.setState({
      loading: true,
      commandsPending: this.state.commandsPending + 1
    })
    try {
      await this.client.powerToggle()
    }
    catch (err) {
      toast.error(err.message)
    }
    this.setState({
      loading: !!(this.state.commandsPending - 1),
      commandsPending: this.state.commandsPending - 1
    })
  }

  setDesiredGrillTemp = async (temperature) => {
    if (!this.canChangeTemp) return
    this.setState({
      loading: true,
      commandsPending: this.state.commandsPending + 1
    })
    try {
      await this.client.setDesiredGrillTemp(temperature)
    }
    catch (err) {
      toast.error(err.message)
    }
    this.setState({
      loading: !!(this.state.commandsPending - 1),
      commandsPending: this.state.commandsPending - 1
    })
  }

  setDesiredFoodTemp = async (temperature) => {
    if (!this.canChangeTemp) return
    this.setState({
      loading: true,
      commandsPending: this.state.commandsPending + 1
    })
    try {
      await this.client.setDesiredFoodTemp(temperature)
    }
    catch (err) {
      toast.error(err.message)
    }
    this.setState({
      loading: !!(this.state.commandsPending - 1),
      commandsPending: this.state.commandsPending - 1
    })
  }

  timerToggle = () => {
    if (this.state.loading) return
    this.setState({ showTimers: !this.state.showTimers })
  }

  historyToggle = () => {
    if (this.state.loading) return
    this.setState({ showHistory: !this.state.showHistory })
  }

  render() {
    return (
      <div className="container">
        <ToastContainer
          position="top-right"
          autoClose={false}
          limit={3}
          newestOnTop
          closeOnClick
          pauseOnHover
          theme="dark"
        />
        {!this.state.socketConnected && <Connecting />}
        <div>
          <HomeControls
            onPowerTouchTap={this.powerToggle}
            onTimersTouchTap={this.timerToggle}
            onHistoryTouchTap={this.historyToggle}
            loading={this.state.loading}
            fanModeActive={this.state.fanModeActive}
            lowPelletAlarmActive={this.state.lowPelletAlarmActive}
            grillConnected={this.state.grillConnected}
            timersOn={this.state.showTimers}
            historyOn={this.state.showHistory}
            powerOn={this.state.isOn} />
        </div>
        <div className="card-container">
          <GrillTemperature
            isEnabled={this.canChangeTemp}
            onSubmit={this.setDesiredGrillTemp}
            desiredGrillTemp={this.state.desiredGrillTemp}
            currentGrillTemp={this.state.currentGrillTemp} />
        </div>
        <div className="card-container">
          <FoodTemperature
            isEnabled={this.canChangeTemp}
            onSubmit={this.setDesiredFoodTemp}
            desiredFoodTemp={this.state.desiredFoodTemp}
            currentFoodTemp={this.state.currentFoodTemp} />
        </div>
        {this.state.showTimers &&
          <div className="card-container">
            <Timers
              isEnabled={true} />
          </div>}
          {this.state.showHistory &&
          <div className="card-container">
            <GrillHistory
              datasets={this.state.datasets}
            />
          </div>}
      </div>
    )
  }
}
