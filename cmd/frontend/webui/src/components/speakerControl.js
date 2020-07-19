import { useEffect, useState } from 'react';
import styled from 'styled-components';
import { getMuteForSpeaker, setMuteForSpeaker } from '../api/service';

function SpeakerControl(props) {
    const [muted, setMuted] = useState();

    useEffect(() => {
        if (props.speakerId) {
            getMuteForSpeaker(props.speakerId).then(isMuted => {
                setMuted(isMuted);
            });
        }
    }, [props.speakerId]);

    function toggleMute(mute) {
        setMuteForSpeaker(props.speakerId, mute).then(() => {
            setMuted(mute);
        })
    }

    const Controls = styled.div`
        display: flex;
        flex-direction: column;
        align-items:center;
        padding-top: 30px;
        padding-bottom: 20px;
        margin-top: 15px;
        background: #474747;
        border-radius: 15px;
        
    `

    return (
        <Controls>
            {muted
                ? <i className="material-icons" onClick={() => { toggleMute(false) }}>volume_off</i>
                : <i className="material-icons" onClick={() => { toggleMute(true) }}>volume_up</i>
            }

        </Controls>
    );
}

export default SpeakerControl;