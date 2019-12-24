import { useEffect, useState } from 'react';
import styled from 'styled-components';
import { getCurrentTrackForSpeaker } from '../api/service';

function NowPlaying(props) {
    const [track, setTrack] = useState();

    useEffect(() => {
        if (props.speakerId) {
            getCurrentTrackForSpeaker(props.speakerId).then(currentTrack => {
                const blob = new Blob([currentTrack.getArtwork()], { type: "image/jpeg" });
                const urlCreator = window.URL || window.webkitURL;
                const imageUrl = urlCreator.createObjectURL(blob);
                currentTrack.artworkUrl = imageUrl;
                setTrack(currentTrack);
            });
        }
    }, [props.speakerId]);

    const Track = styled.div`
        display: flex;
        flex-direction: column;
        align-items:center;
        padding-top: 30px;
        padding-bottom: 20px;
        margin-top: 15px;
        background: #474747;
        border-radius: 30px;
        width: 300px;
    `
    const AlbumArt = styled.img`
       height: 200px;
       width: 200px;
    `
    const TrackDetails = styled.div`
       display: flex;
       flex-direction: column;
       align-items: center;
       justify-content: center;
       padding-top: 10px;
    `
    return (
        <Track>
            <AlbumArt src={track && track.artworkUrl ? track.artworkUrl : undefined}></AlbumArt>
            <TrackDetails>
                <span>{track && track.getTitle()}</span>
                <span>{track && track.getArtist()} -- {track && track.getAlbum()}</span>
            </TrackDetails>
        </Track>
    );
}

export default NowPlaying;