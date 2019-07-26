import { from, timer } from 'rxjs';
import { switchMap } from 'rxjs/operators';
import { GetSpeakersRequest, GetZonesRequest } from './management_pb.js';
import { BobcaygeonManagementPromiseClient } from './management_grpc_web_pb.js';

const DEFAULT_POLL_INTERVAL = 15000;

const mgmtService = new BobcaygeonManagementPromiseClient(`http://${window.location.hostname}:9211`);

// helper function to poll APIs
// from: https://codewithhugo.com/better-http-polling-with-rxjs-5/
function poll(fetchFn, pollInterval = DEFAULT_POLL_INTERVAL) {
    // inverval is an observable that emits a count every `pollInterval` ms
    // what we do is use switch map to convert it from the integer that is
    // emitted, to something useful.  `from` will then execute `fetchFn` and
    // convert the result to an observable
    return timer(0, pollInterval).pipe(
        switchMap(() => from(fetchFn()))
    );
}

// returns an observable wrapping the getSpeakers API
export const speakerListObservable = () => {
    const request = new GetSpeakersRequest();
    // poll for now, maybe switch to grpc streaming?
    return poll(() => {
        return mgmtService.getSpeakers(request, {});
    })
}

// returns an observable wrapping the getSpeakers API
export const zoneListObservable = () => {
    const request = new GetZonesRequest();
    // poll for now, maybe switch to grpc streaming?
    return poll(() => {
        return mgmtService.getZones(request, {});
    })
}