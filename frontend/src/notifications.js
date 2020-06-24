import Vue from 'vue'
import VueNotifications from 'vue-notifications'
import miniToastr from 'mini-toastr'

// Here we setup messages output to `mini-toastr`
function toast ({title, message, type, timeout, cb}) {
    return miniToastr[type](message, title, timeout, cb)
}
VueNotifications.config.timeout = 4000;
// Activate plugin
Vue.use(VueNotifications, {
    success: toast,
    error: toast,
    info: toast,
    warn: toast
}); // VueNotifications have auto install but if we want to specify options we've got to do it manually.
miniToastr.init({types: {
    success: 'success',
    error: 'error',
    info: 'info',
    warn: 'warn'
}});


export default {
    error(m, b, s) {
        VueNotifications.error({title: 'Unexpected server error', message: 'Unexpected server error occurred on '+m+' '+b + ' ' + s})
    },
    info(title, message){
        VueNotifications.info({title: title, message: message})
    }
}