
import { useState, useEffect } from 'react';
import styled from 'styled-components';
import { getSpeakers } from '../api/service';
import SpeakerZoneSelect from './speakerZoneSelect';
import NowPlaying from './nowPlaying';
import SpeakerDetails from './speakerDetails';

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
            <SpeakerDetails selectedSpeaker={selectedSpeaker} speakerUpdated={speakerUpdated}></SpeakerDetails>
        </div>
    );
}


export default App;