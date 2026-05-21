import React, { Component } from 'react'
import PropTypes from 'prop-types'
import { Line } from 'react-chartjs-2'
import Card from '@mui/material/Card'
import CardMedia from '@mui/material/CardMedia'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import logo from './logo.png'

// 30-minute streaming window (was set via Chart.defaults.global in v2;
// in v3+ realtime config lives on the scale itself).
const STREAM_DURATION_MS = 30 * 60 * 1000

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
                <Line
                    data={{ datasets: this.props.datasets }}
                    options={{
                        scales: {
                            x: {
                                type: 'realtime',
                                time: {
                                    unit: 'minute',
                                    // 24h "HH:mm" instead of the adapter's
                                    // default "h:mm a" — wider AM/PM glyphs
                                    // crowded the axis on smaller viewports.
                                    displayFormats: { minute: 'HH:mm' }
                                },
                                ticks: {
                                    // Show ~one label every few minutes
                                    // rather than every minute. Chart.js
                                    // honors autoSkip by default; this is
                                    // the explicit upper bound.
                                    maxTicksLimit: 8
                                },
                                realtime: { duration: STREAM_DURATION_MS }
                            },
                            y: { type: 'linear' }
                        },
                        plugins: {
                            tooltip: { mode: 'nearest', intersect: false }
                        },
                        // In v2 'hover' was top-level; in v3+ it became 'interaction'.
                        interaction: { mode: 'nearest', intersect: false }
                    }}
                />
            </Card>
        )
    }
}

GrillHistory.propTypes = {
    datasets: PropTypes.array
}
