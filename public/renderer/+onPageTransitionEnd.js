import bus, {SET_LOADING} from "../common/bus.js";

export { onPageTransitionEnd }

function onPageTransitionEnd(pageContext) {
    if (typeof window == "undefined") {
        //console.log("onPageTransitionEnd Application is on server side", pageContext);
        return undefined
    } else {
        //console.log("onPageTransitionEnd Application is on client side", pageContext);
        bus.emit(SET_LOADING, false)
        return undefined
    }

}