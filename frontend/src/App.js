import React, {  useEffect } from "react";
import "./header.css"
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
                        <img src="https://png.pngtree.com/element_origin_min_pic/16/09/11/1057d4c846189bf.jpg"
                             alt="Logo" className="logo__pic"/>
                    </div>
                    <nav className="menu">
                        <Link className="menu__item" to="/">Главная</Link>
                        <Link className="menu__item" to="/public/about">About</Link>
                        <Link className="menu__item" to="/public/chat">Chats</Link>
                    </nav>
                    <div className="login">
                        <button className="login__btn">Войти</button>
                    </div>
                </div>
            </div>
        </header>

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