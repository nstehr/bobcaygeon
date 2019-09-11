import { useEffect, useState } from 'react';
import styled from 'styled-components';
import { changeDisplayNameForSpeaker } from '../api/service';

function SpeakerDetails(props) {
    const [displayName, setDisplayName] = useState("");
    const [editMode, setEditMode] = useState(false);
    useEffect(() => {
        if (props.selectedSpeaker) {
            setDisplayName(props.selectedSpeaker && props.selectedSpeaker.getDisplayname() ? props.selectedSpeaker.getDisplayname() : props.selectedSpeaker.getId())
        }
    }, [props.selectedSpeaker]);

    const handleSubmit = (event) => {
        event.preventDefault();
        changeDisplayNameForSpeaker(props.selectedSpeaker.getId(), displayName, false).then(() => {
            if (props.speakerUpdated) {
                props.speakerUpdated();
            }
        });

    }

    const handleChange = (event) => {
        setDisplayName(event.target.value);
    }

    const Details = styled.div`
    display: flex;
    flex-direction: column;
    align-items:center;
    padding-top: 30px;
    padding-bottom: 20px;
    margin-top: 15px;
    background: #474747;
    border-radius: 15px;
    `

    const Edit = styled.div`
    display: flex;
    
    i {
        padding-right: 15px;
        color: ${props => props.active ? "white" : "palevioletred"};
        
    }
    `
    const EditForm = styled.div`
    display: flex;
    input[type="submit"]  
    {
       margin-left: 15px;
    }
    input[type="text"]  
    {
        border: 0;
        outline: 0;
        background: transparent;
        border-bottom: 1px solid #212121;
        color: #fff;
    }
    `

    return (
        <Details>
            <Edit active={!editMode}>
                <i className="material-icons" onClick={() => { setEditMode(!editMode) }}>edit</i>
            </Edit>
            {editMode && (
                <EditForm>
                    < form onSubmit={handleSubmit}>
                        <input type="text" id="name" value={displayName} onChange={handleChange}></input>
                        <input type="submit" value="Update" />
                    </form>
                </EditForm>
            )}
        </Details >
    );
}

export default SpeakerDetails;