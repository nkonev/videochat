import axios from "axios";
import {setProfile} from "./actions";

export const getProfile = (dispatch) => {
    return axios.get(`/api/profile`)
        .then(value1 => {
            return dispatch(setProfile(value1.data));
        })

};