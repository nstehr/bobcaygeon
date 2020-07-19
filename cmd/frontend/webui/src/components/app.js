
import { useState, useEffect } from 'react';
import styled from 'styled-components';
import { getSpeakers } from '../api/service';
import SpeakerZoneSelect from './speakerZoneSelect';
import NowPlaying from './nowPlaying';
import SpeakerDetails from './speakerDetails';
import SpeakerControl from './speakerControl';

function App() {
    const [speakers, setSpeakers] = useState([]);
    const [selectedSpeaker, setSelectedSpeaker] = useState();

    useEffect(() => {
        getSpeakers().then((speakers) => {
            setSpeakers(speakers);
        });
    }, []);

    const speakerSelected = (speaker) => {
        setSelectedSpeaker(speaker);
    };

    const speakerUpdated = () => {
        getSpeakers().then((speakers) => {
            setSpeakers(speakers);
        });
    }

    return (
        <div>
            <SpeakerZoneSelect speakers={speakers} speakerSelected={speakerSelected} />
            <NowPlaying speakerId={selectedSpeaker ? selectedSpeaker.getId() : undefined}></NowPlaying>
            <SpeakerControl speakerId={selectedSpeaker ? selectedSpeaker.getId() : undefined}></SpeakerControl>
            <SpeakerDetails selectedSpeaker={selectedSpeaker} speakerUpdated={speakerUpdated}></SpeakerDetails>
        </div>
    );
}


export default App;