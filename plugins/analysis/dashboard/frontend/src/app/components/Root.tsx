import * as React from 'react';
import {inject, observer} from "mobx-react";
import NodeStore from "app/stores/NodeStore";
import Navbar from "react-bootstrap/Navbar";
import Nav from "react-bootstrap/Nav";
import {Autopeering} from "app/components/Autopeering";
import {RouterStore} from 'mobx-react-router';
import {Redirect, Route, Switch} from 'react-router-dom';
import {LinkContainer} from 'react-router-bootstrap';
import FPC from "app/components/FPC";
import Conflict from "app/components/Conflict";

interface Props {
    history: any;
    routerStore?: RouterStore;
    nodeStore?: NodeStore;
}

@inject("nodeStore")
@inject("routerStore")
@observer
export class Root extends React.Component<Props, any> {
    renderDevTool() {
        if (process.env.NODE_ENV !== 'production') {
            const DevTools = require('mobx-react-devtools').default;
            return <DevTools/>;
        }
    }

    componentDidMount(): void {
        this.props.nodeStore.connect();
    }

    render() {
        return (
            <div className="container">
                <Navbar expand="lg" bg="light" variant="light" className={"mb-4"}>
                    <Navbar.Brand>GoShimmer analyser</Navbar.Brand>
                    <Nav className="mr-auto">
                        <LinkContainer to="/dashboard">
                            <Nav.Link>Autopeering</Nav.Link>
                        </LinkContainer>
                        <LinkContainer to="/fpc-example">
                            <Nav.Link>
                                Consensus
                            </Nav.Link>
                        </LinkContainer>
                    </Nav>
                </Navbar>
                <Switch>
                    <Route exact path="/autopeering" component={Autopeering}/>
                    <Route exact path="/fpc-example" component={FPC}/>
                    <Route exact path="/fpc-example/conflict/:id" component={Conflict}/>
                    <Redirect to="/autopeering"/>
                </Switch>
                {this.props.children}
                {this.renderDevTool()}
            </div>
        );
    }
}