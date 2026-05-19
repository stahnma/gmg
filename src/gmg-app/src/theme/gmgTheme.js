import { createTheme } from '@mui/material/styles'
import { green, grey } from '@mui/material/colors'

// Port of the old material-ui 0.20 darkBaseTheme + custom palette:
//   textColor: grey200, primary1Color: grey50, accent1Color: green700,
//   accent2Color: grey800, canvas: #303030
const theme = createTheme({
  palette: {
    mode: 'dark',
    primary: { main: green[700] },
    secondary: { main: grey[50] },
    background: { default: '#303030', paper: grey[800] },
    text: { primary: grey[200] },
  },
})

export default theme
