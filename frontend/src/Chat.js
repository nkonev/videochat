import React, {useState, useEffect} from 'react';
import axios from 'axios'
import {makeStyles, withStyles} from '@material-ui/core/styles';
import List from '@material-ui/core/List';
import ListItem from '@material-ui/core/ListItem';
import ListItemText from '@material-ui/core/ListItemText';
import Button from '@material-ui/core/Button';
import Grid from '@material-ui/core/Grid';
import Fab from '@material-ui/core/Fab';
import AddIcon from '@material-ui/icons/Add';
import Box from '@material-ui/core/Box';
import { green, common } from '@material-ui/core/colors';
import Breadcrumbs from '@material-ui/core/Breadcrumbs';
import Link from '@material-ui/core/Link';

const circleCheckRadius = 34;
const useStyles = makeStyles(theme => ({
    appHeader: {
        'background-color': '#282c34',
        display: 'flex',
        'flex-direction': 'column',
        'align-items': 'center',
        'justify-content': 'center',
        'font-size': 'calc(10px + 2vmin)',
        color: 'white',
        'word-wrap': 'break-word',
        'font-family': 'monospace',
    },
    root: {
        width: '100%',
        backgroundColor: theme.palette.background.paper,
    },
    fabAddButton: {
        position: 'absolute',
        zIndex: 1,
        bottom: 30,
        right: 30,
        margin: '0 auto',
    },
    fabRestoreButton: {
        position: 'absolute',
        zIndex: 1,
        bottom: 30,
        left: 30,
        margin: '0 auto',
    },
    paper: {
        position: 'absolute',
        width: 400,
        backgroundColor: theme.palette.background.paper,
        border: '2px solid #000',
        boxShadow: theme.shadows[5],
        padding: theme.spacing(2),
    },
    confirm: {
        position: 'absolute',
        backgroundColor: theme.palette.background.paper,
        border: '2px solid #000',
        boxShadow: theme.shadows[5],
        padding: theme.spacing(2),
    },
    typography: {
        padding: theme.spacing(2),
    },
    buttonProgress: {
        color: green[500],
        position: 'absolute',
        top: '50%',
        left: '50%',
        marginTop: -circleCheckRadius,
        marginLeft: -circleCheckRadius,
    },
}));

function getModalStyle() {
    const top = 50;
    const left = 50;

    return {
        top: `${top}%`,
        left: `${left}%`,
        transform: `translate(-${top}%, -${left}%)`,
    };
}

const GreenButton = withStyles(theme => ({
    root: {
        color: common[500],
        backgroundColor: green[500],
        '&:hover': {
            backgroundColor: green[700],
        },
    },
}))(Button);

function Chat() {
    // state
    const [chats, setChats] = useState([]);
    const [modalStyle] = React.useState(getModalStyle);

    const fetchData = () => {
        axios.get(`/api/chat`)
            .then(message => {
                setChats(message.data);
            });
    };

    useEffect(() => {
        fetchData();
    }, []);

    const classes = useStyles();

    return (
        <div className="App">
            <div className={classes.root}>
                <header className={classes.appHeader}>
                    <div className="header-text">Videochat</div>
                </header>
                <Breadcrumbs aria-label="breadcrumb">
                    <Link color="inherit" href="/">
                        Chats
                    </Link>
                    <Link color="inherit" href="/">
                        Current chat
                    </Link>
                </Breadcrumbs>
                <List className="list-db-connections">
                    {chats.map((value, index) => {
                        return (
                            <ListItem key={value.id} button>

                                <Grid container spacing={1} direction="row">
                                    <Grid container item xs alignItems="center" spacing={1} className="downloadable-clickable">
                                        <ListItemText>
                                            <Box fontFamily="Monospace" className="list-element">
                                                {value.name}
                                            </Box>
                                        </ListItemText>
                                    </Grid>

                                    <Grid container item xs={2} direction="row"
                                          justify="flex-end"
                                          alignItems="center" spacing={1}>
                                        <Grid item>
                                            <Button variant="contained" color="primary">
                                                Share
                                            </Button>
                                        </Grid>
                                        <Grid item>
                                            <Button variant="contained" color="secondary">
                                                Delete
                                            </Button>
                                        </Grid>
                                    </Grid>
                                </Grid>
                            </ListItem>
                        )
                    })}
                </List>

                <Fab color="primary" aria-label="add" className={classes.fabAddButton}>
                    <AddIcon className="fab-add"/>
                </Fab>
            </div>

        </div>
    );
}

export default (Chat);
