import axios from "axios";

export default () => {
    return {
        methods: {
            triggerUsesStatusesEvents(userIdsJoined, signal) {
                axios.put(`/api/aaa/user/request-for-online`, null, {
                    params: {
                        userId: userIdsJoined
                    },
                    signal: signal
                }).then(()=>{
                    axios.put("/api/video/user/request-in-video-status", null, {
                        params: {
                            userId: userIdsJoined
                        },
                        signal: signal
                    });
                })
            }
        }
    }
}