import React, { Component } from 'react'
import PropTypes from 'prop-types'
import { Line } from 'react-chartjs-2'
import Card from '@mui/material/Card'
import CardMedia from '@mui/material/CardMedia'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import 'chartjs-plugin-streaming'
import logo from './logo.png'

export default class GrillHistory extends Component {
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
                        <Typography variant="h6">Temperature History</Typography>
                        <Typography variant="body2">View grilling temperature history.</Typography>
                    </Box>
                </Box>
                <Line data={{
                    datasets: this.props.datasets
                }}
                    options={{
                        scales: {
                            xAxes: [{
                                type: 'realtime',
                                time: { unit: 'minute' }
                            }],
                            tooltips: {
                                mode: 'nearest',
                                intersect: false
                            },
                            hover: {
                                mode: 'nearest',
                                intersect: false
                            },
                        }
                    }}
                />
            </Card>
        )
    }
}

GrillHistory.propTypes = {
    datasets: PropTypes.array
}
