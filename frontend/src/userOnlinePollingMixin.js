import axios from "axios";

const pollingInterval = 2000;

export default () => {
    let intervalId;

    return  {
        methods: {
            startPolling(participantsProvider, handler) {
                intervalId = setInterval(()=>{
                    axios.get(`/api/chat/online`, {
                        params: {
                            participantIds: participantsProvider().reduce((f, s) => `${f},${s}`)
                            // participantIds: [1,2,3].reduce((f, s) => `${f},${s}`)
                        }
                    }).then(value => {
                        handler(value.data)
                    })
                }, pollingInterval);
            },
            stopPolling() {
                clearInterval(intervalId)
            },
        },

    }
}