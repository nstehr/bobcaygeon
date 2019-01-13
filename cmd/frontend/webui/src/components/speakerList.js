import { useEffect, useState } from 'react';
import { speakerListObservable } from '../api/observables';
import styled from 'styled-components'

// list of speaker elements
function SpeakerList() {
    const [speakers, setSpeakers] = useState([]);
    useEffect(() => {
        // retrieve an observable to the speaker list
        const speakerList = speakerListObservable();
        // create a subscription to the observable
        const sub = speakerList.subscribe(resp => {
            // update the state when we get a new list of speakers
            setSpeakers(resp.getSpeakersList());
        })
        // returns the function that will be called when our component is destroyed
        // in our case it is to unsubscribe
        return () => {
            sub.unsubscribe();
        };
    }, []);

    return (
        <div>
            <h2>Speakers</h2>
            {speakers.map(item => <SpeakerItem key={item.getId()} speaker={item}></SpeakerItem>)}
        </div>
    );
}

// Individual row in the list
// keeping it inline for now
function SpeakerItem(props) {
    const SpeakerRow = styled.div`
        display: flex;
        flex-direction: row;
        align-items: center;
        min-width: 300px;
        i:last-child {
            margin-left: auto;
          }
        span {
            margin-left: 15px;
        }
    `
    const speaker = props.speaker;
    return (
        <SpeakerRow>
            <i className="material-icons">speaker</i>
            <span>{speaker.getDisplayname() ? speaker.getDisplayname() : speaker.getId()}</span>
            <i className="material-icons">edit</i>
        </SpeakerRow>
    );
}

export default SpeakerList;