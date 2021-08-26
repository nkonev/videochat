import axios from "axios";

const pollingInterval = 2000;

export default () => {
    let intervalId;

    return  {
        methods: {
            startPolling(participantsProvider, handler) {
                intervalId = setInterval(()=>{
                    const participants = participantsProvider();
                    if (!participants || participants.length == 0) {
                        console.debug("Participants are empty or participantsProvider returned equal null, invoking handler with empty array");
                        handler([]);
                        return;
                    }
                    console.debug("Participants are non-empty, invoking axios");
                    axios.get(`/api/chat/online`, {
                        params: {
                            participantIds: participants.reduce((f, s) => `${f},${s}`)
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
