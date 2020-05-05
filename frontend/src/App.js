import React, {  useEffect } from "react";
import "./header.css"
import {
    BrowserRouter as Router,
    Switch,
    Route,
    Link
} from "react-router-dom";
import Chat from "./Chat";
import Login from "./Login";

/**
 * Main landing page user starts interaction from
 * @returns {*}
 * @constructor
 */

function App() {

    // https://ru.reactjs.org/docs/hooks-effect.html
    useEffect(() => {
        function showHeader() {
            console.log("Called showHeader");
            var header = document.querySelector('.header');
            if(window.pageYOffset > 200){
                header.classList.add('header_fixed');
            } else{
                header.classList.remove('header_fixed');
            }
        }
        console.log("Add header scroll listener");
        window.onscroll = showHeader;

        return function cleanup() {
            console.log("Removing header scroll listener");
            window.onscroll = null;
        };
    });

    return (
        <Router>
        <header className="header">
            <div className="wrapper">
                <div className="row">
                    <div className="logo">
                        <img src="/public/assets/pandas.jpg"
                             alt="Logo" className="logo__pic"/>
                    </div>
                    <nav className="menu">
                        <Link className="menu__item" to="/">Главная</Link>
                        <Link className="menu__item" to="/about">About</Link>
                        <Link className="menu__item" to="/chat">Chats</Link>
                    </nav>
                    <div className="login">
                        <Link className="menu__item login__btn" to="/login">Войти</Link>
                    </div>
                </div>
            </div>
        </header>

        {/* A <Switch> looks through its children <Route>s and
        renders the first one that matches the current URL. */}
        <Switch>
            <Route path="/login">
                <Login />
            </Route>
            <Route path="/about">
                <About />
            </Route>
            <Route path="/chat">
                <Chat />
            </Route>
            <Route path="/">
                <Home />
            </Route>

        </Switch>

        </Router>
    );
}

function Home() {
    return <div>
        <h2>Welcome</h2>
        <div>Best place to chatting with your friends.</div>
    </div>;
}

function About() {
    return <div>
        This is best application for video calls and text messaging. We was hardworking.
    </div>;
}

export default (App);