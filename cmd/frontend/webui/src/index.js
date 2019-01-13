import React from 'react';
import ReactDOM from 'react-dom';
import { createGlobalStyle } from 'styled-components';

// Import Components
import Container from './components/container';
import Header from './components/header';
import SpeakerList from './components/speakerList';

// Global Style
const GlobalStyle = createGlobalStyle`
  body {
    background: #212121;
    color: #fff;
    padding: 1em;
    line-height: 1.8em;
		font-size: 15;
    -webkit-font-smoothing: antialiased;
    text-rendering: optimizeSpeed;
    word-wrap: break-word
  }
`;

// Render page
ReactDOM.render(
    <Container>
        <Header>Bobcaygeon 🎸</Header>
        <p>Manage your speakers and speaker zones</p>
        <SpeakerList />
        <GlobalStyle />
    </Container>,
    document.getElementById('root')
);