import bus, {SET_LOADING} from "../common/bus.js";

export { onPageTransitionStart }

function onPageTransitionStart(pageContext) {
    if (typeof window == "undefined") {
        //console.log("+onPageTransitionStart Application is on server side", pageContext);
        return undefined
    } else {
        //console.log("+onPageTransitionStart Application is on client side", pageContext);
        bus.emit(SET_LOADING, true)
        return undefined
    }

}