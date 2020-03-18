import React from "react";
import {
    BrowserRouter as Router,
    Switch,
    Route,
    Link
} from "react-router-dom";
import Chat from "./Chat";

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
        <Router>
            <div>
                <nav>
                    <ul>
                        <li>
                            <Link to="/">Home</Link>
                        </li>
                        <li>
                            <Link to="/public/about">About</Link>
                        </li>
                        <li>
                            <Link to="/public/chat">Chats</Link>
                        </li>
                    </ul>
                </nav>

                {/* A <Switch> looks through its children <Route>s and
            renders the first one that matches the current URL. */}
                <Switch>
                    <Route path="/public/about">
                        <About />
                    </Route>
                    <Route path="/public/chat">
                        <Chat />
                    </Route>
                    <Route path="/">
                        <Home />
                    </Route>
                </Switch>
            </div>
        </Router>
    );
}

function Home() {
    return <div>
        <h2>Welcome</h2>
        <text>Best place to chatting with your friends.</text>
    </div>;
}

function About() {
    return <div>
        <text>This is best application for video calls and text messaging. We was hardworking.</text>
    </div>;
}

export default (App);