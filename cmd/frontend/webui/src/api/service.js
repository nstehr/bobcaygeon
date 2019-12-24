import { GetSpeakersRequest, GetZonesRequest, GetTrackRequest, SetSpeakerDisplayNameRequest } from './management_pb.js';
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