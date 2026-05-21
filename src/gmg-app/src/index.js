import React from 'react';
import { createRoot } from 'react-dom/client';
import './index.css';
import './chartSetup';
import App from './components/App/index';

createRoot(document.getElementById('root')).render(<App />);
