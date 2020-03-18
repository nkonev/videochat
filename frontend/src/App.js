import React from "react";

/**
 * Main landing page user starts interaction from
 * @returns {*}
 * @constructor
 */

function port() {
    if (location.port) {
        return ":"+location.port
    } else {
        return ""
    }
}

function App() {
    return (
        <div className="App">
            <ul>
                {/* Navbar */}
                <li>Pricing</li>
                {/* TODO fix login */}
                <li><a href={"auth.site.local"}>Login</a></li>
                <li><a href={"//site.local"+port()+"/chat"}>My chats</a></li>
            </ul>
        </div>
    )
}

export default (App);