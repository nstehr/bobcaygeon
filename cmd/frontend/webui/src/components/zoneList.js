import { useEffect, useState } from 'react';
import { zoneListObservable } from '../api/observables';
import styled from 'styled-components'

// list of zone elements
function ZoneList() {
    const [zones, setZones] = useState([]);
    useEffect(() => {
        // retrieve an observable to the zone list
        const zoneList = zoneListObservable();
        // create a subscription to the observable
        const sub = zoneList.subscribe(resp => {
            // update the state when we get a new list of zones
            setZones(resp.getZonesList());
        })
        // returns the function that will be called when our component is destroyed
        // in our case it is to unsubscribe
        return () => {
            sub.unsubscribe();
        };
    }, []);

    const AddZone = styled.div`
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

    return (
        <div>
            <h2>Zones</h2>
            <div>
                {zones.map(item => <ZoneItem key={item.getId()} zone={item}></ZoneItem>)}
            </div>
            <AddZone>
                <span>Create Zone</span>
                <i className="material-icons">add_circle</i>
            </AddZone>
        </div>
    );
}

// Individual row in the list
// keeping it inline for now
function ZoneItem(props) {
    const ZoneRow = styled.div`
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
    const zone = props.zone;
    return (
        <ZoneRow>
            <i className="material-icons">speaker_group</i>
            <span>{zone.getDisplayname()}</span>
        </ZoneRow>
    );
}

export default ZoneList;