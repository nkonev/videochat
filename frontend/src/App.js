import React from "react";
import {
    BrowserRouter as Router,
    Switch,
    Route,
    Link
} from "react-router-dom";

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
                        <Chats />
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
    return <h2>Home</h2>;
}

function About() {
    return <h2>About</h2>;
}

function Chats() {
    return <h2>Chats</h2>;
}

export default (App);