import React from 'react';
import ReactDOM from 'react-dom';
import { createGlobalStyle } from 'styled-components';

// Import Components
import Container from './components/container';
import Header from './components/header';

import { GetSpeakersRequest } from './management_pb.js';
import { BobcaygeonManagementClient } from './management_grpc_web_pb.js';

// TODO: testing grpc calls, will move
const mgmtService = new BobcaygeonManagementClient('http://localhost:9211');
const request = new GetSpeakersRequest();
mgmtService.getSpeakers(request, {}, function (err, response) {
    console.log(err);
    console.log(response.getSpeakersList());
});

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
        <Header>Hello World 🎸</Header>
        <p>Example site using Styled React Boilerplate</p>
        <GlobalStyle />
    </Container>,
    document.getElementById('root')
);