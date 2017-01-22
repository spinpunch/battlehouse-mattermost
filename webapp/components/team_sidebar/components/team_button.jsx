// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

import Constants from 'utils/constants.jsx';

import React from 'react';
import {Link} from 'react-router/es6';
import {Tooltip, OverlayTrigger} from 'react-bootstrap';

export default class TeamButton extends React.Component {
    constructor(props) {
        super(props);

        this.handleDisabled = this.handleDisabled.bind(this);
    }

    handleDisabled(e) {
        e.preventDefault();
    }

    render() {
        let teamClass = this.props.active ? 'active' : '';
        const btnClass = this.props.btnClass;
        const disabled = this.props.disabled ? 'team-disabled' : '';
        const handleClick = (this.props.active || this.props.disabled) ? this.handleDisabled : null;
        let badge;

        if (!teamClass) {
            teamClass = this.props.unread ? 'unread' : '';

            if (this.props.mentions) {
                badge = (
                    <span className='badge pull-right small'>{this.props.mentions}</span>
                );
            }
        }

        let btn;
        let content = this.props.content;
        if (!content) {
            content = (
                <div className='team-btn__initials'>
                    {this.props.displayName.substring(0, 2)}
                    <div className='team-btn__content'>
                        {this.props.displayName}
                    </div>
                </div>
            );
        }
        if (this.props.isMobile) {
            btn = (
                <div className={'team-btn ' + btnClass}>
                    {badge}
                    {content}
                </div>
            );
        } else {
            btn = (
                <OverlayTrigger
                    delayShow={Constants.OVERLAY_TIME_DELAY}
                    placement={this.props.placement}
                    overlay={
                        <Tooltip id={`tooltip-${this.props.url}`}>
                            {this.props.tip}
                        </Tooltip>
                    }
                >
                    <div className={'team-btn ' + btnClass}>
                        {badge}
                        {content}
                    </div>
                </OverlayTrigger>
            );
        }

        return (
            <div
                className={`team-container ${teamClass}`}
            >
                <Link
                    className={disabled}
                    to={this.props.url}
                    onClick={handleClick}
                >
                    {btn}
                </Link>
            </div>
        );
    }
}

TeamButton.defaultProps = {
    btnClass: '',
    tip: '',
    placement: 'right',
    active: false,
    disabled: false,
    unread: false,
    mentions: 0
};

TeamButton.propTypes = {
    btnClass: React.PropTypes.string,
    url: React.PropTypes.string.isRequired,
    displayName: React.PropTypes.string,
    content: React.PropTypes.node,
    tip: React.PropTypes.node.isRequired,
    active: React.PropTypes.bool,
    disabled: React.PropTypes.bool,
    isMobile: React.PropTypes.bool,
    unread: React.PropTypes.bool,
    mentions: React.PropTypes.number,
    placement: React.PropTypes.oneOf(['left', 'right', 'top', 'bottom'])
};
