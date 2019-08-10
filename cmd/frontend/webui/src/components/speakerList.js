import { useEffect, useState } from 'react';
import { speakerListObservable, getCurrentTrackForSpeakerObservable } from '../api/observables';
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
        justify-content: space-between;
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
            <SpeakerTrack speaker={speaker}></SpeakerTrack>
        </SpeakerRow>
    );
}

// TODO: hack to do some testing....
function SpeakerTrack(props) {
    const [track, setTrack] = useState([]);
    useEffect(() => {
        // retrieve an observable to the speaker list
        const track = getCurrentTrackForSpeakerObservable(props.speaker.getId());
        // create a subscription to the observable
        const sub = track.subscribe(resp => {
            var blob = new Blob([resp.getArtwork()], { type: "image/jpeg" });
            var urlCreator = window.URL || window.webkitURL;
            var imageUrl = urlCreator.createObjectURL(blob);
            resp.artworkUrl = imageUrl;
            setTrack(resp);
        })
        // returns the function that will be called when our component is destroyed
        // in our case it is to unsubscribe
        return () => {
            sub.unsubscribe();
        };
    }, []);
    const Track = styled.div`
        img {
            height:32px;
            width:32px;
        }
    `
    return (
        <Track>
            <img src={track.artworkUrl}></img>
        </Track>
    );
}

export default SpeakerList;