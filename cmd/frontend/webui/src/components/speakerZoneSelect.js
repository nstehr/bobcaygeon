import { useEffect, useState } from 'react';
import styled from 'styled-components';

// Select box fox selecting speakers or zones
function SpeakerZoneSelect(props) {
    const [selectedSpeaker, setSelectedSpeaker] = useState();

    useEffect(() => {
        if (!selectedSpeaker && props.speakers.length > 0) {
            setSelectedSpeaker(props.speakers[0]);
        }
    }, [[props.speakers]])

    useEffect(() => {
        props.speakerSelected(selectedSpeaker);
    }, [selectedSpeaker]);

    const selectedHandler = (event) => {
        const selectedId = event.target.value;
        const clickedSpeaker = props.speakers.filter(speaker => { return speaker.getId() === selectedId });
        if (clickedSpeaker.length === 1) {
            setSelectedSpeaker(clickedSpeaker[0]);
        }

    }
    const SpeakerZoneSelector = styled.div`
        display: flex;
        flex-direction: row;
        justify-content: center;
        select {
            min-width: 250px;
        }
    `
    return (
        <SpeakerZoneSelector>
            <select onChange={selectedHandler} value={selectedSpeaker && selectedSpeaker.getId()}>
                <optgroup label="Speakers">
                    {props.speakers && props.speakers.map(item => <option value={item.getId()} key={item.getId()}>{item.getDisplayname() ? item.getDisplayname() : item.getId()}</option>)}
                </optgroup>
            </select>
        </SpeakerZoneSelector>
    );
}



export default SpeakerZoneSelect;