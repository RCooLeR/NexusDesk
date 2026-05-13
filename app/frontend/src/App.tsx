import {useEffect, useState} from 'react';
import {GetStartupState} from '../wailsjs/go/main/App';
import {fallbackState} from './data/startupState';
import {NexusDeskShell} from './features/shell/NexusDeskShell';
import type {StartupState} from './types';
import './App.css';

function App() {
    const [state, setState] = useState<StartupState>(fallbackState);

    useEffect(() => {
        Promise.resolve()
            .then(() => GetStartupState())
            .then(setState)
            .catch(() => setState(fallbackState));
    }, []);

    return <NexusDeskShell state={state} />;
}

export default App;
