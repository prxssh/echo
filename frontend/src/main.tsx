import React from 'react';
import { createRoot } from 'react-dom/client';
import './style.css';
import './styles/global.css';
import App from './App';

const container = document.getElementById('root');

const root = createRoot(container!);

// Lock the app to Dark OLED theme
document.documentElement.setAttribute('data-theme', 'dark-oled');

root.render(
    <React.StrictMode>
        <App />
    </React.StrictMode>
);
