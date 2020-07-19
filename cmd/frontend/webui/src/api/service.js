import { GetSpeakersRequest, GetZonesRequest, GetTrackRequest, SetSpeakerDisplayNameRequest, GetMuteRequest, SetMuteRequest } from './management_pb.js';
import { BobcaygeonManagementPromiseClient } from './management_grpc_web_pb.js';

const mgmtService = new BobcaygeonManagementPromiseClient(`http://${window.location.hostname}:9211`);

// returns a a list of speakers
export const getSpeakers = async () => {
    const request = new GetSpeakersRequest();
    const speakerResp = await mgmtService.getSpeakers(request, {});
    return speakerResp.getSpeakersList();
}

// returns a list of zones
export const getZones = () => {
    const request = new GetZonesRequest();
    return mgmtService.getZones(request, {});
}

// returns the current track for a given speaker
export const getCurrentTrackForSpeaker = async (speakerId) => {
    const request = new GetTrackRequest();
    request.setSpeakerid(speakerId);
    const trackResp = await mgmtService.getCurrentTrack(request);
    return trackResp;
}

export const changeDisplayNameForSpeaker = async (speakerId, displayName, updateBroadcast) => {
    const request = new SetSpeakerDisplayNameRequest();
    request.setSpeakerid(speakerId);
    request.setDisplayname(displayName);
    request.setUpdatebroadcast(updateBroadcast);
    const changeRequest = await mgmtService.setDisplayNameForSpeaker(request);
    return changeRequest;

}

export const getMuteForSpeaker = async (speakerId) => {
    const request = new GetMuteRequest();
    request.setSpeakerid(speakerId);
    const isMutedResp = await mgmtService.getMuteForSpeaker(request);
    console.log(isMutedResp);
    return isMutedResp.getIsmuted();
}

export const setMuteForSpeaker = async (speakerId, mute) => {
    const request = new SetMuteRequest();
    request.setSpeakerid(speakerId);
    request.setIsmuted(mute);
    const isMuted = await mgmtService.setMuteForSpeaker(request);
    return isMuted;
}