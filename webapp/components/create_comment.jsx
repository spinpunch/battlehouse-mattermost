// Copyright (c) 2015 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

import $ from 'jquery';
import ReactDOM from 'react-dom';
import AppDispatcher from '../dispatcher/app_dispatcher.jsx';
import * as ChannelActions from 'actions/channel_actions.jsx';
import EmojiStore from 'stores/emoji_store.jsx';
import UserStore from 'stores/user_store.jsx';
import PostDeletedModal from './post_deleted_modal.jsx';
import PostStore from 'stores/post_store.jsx';
import PreferenceStore from 'stores/preference_store.jsx';
import MessageHistoryStore from 'stores/message_history_store.jsx';
import Textbox from './textbox.jsx';
import MsgTyping from './msg_typing.jsx';
import FileUpload from './file_upload.jsx';
import FilePreview from './file_preview.jsx';
import * as Utils from 'utils/utils.jsx';
import * as UserAgent from 'utils/user_agent.jsx';
import * as GlobalActions from 'actions/global_actions.jsx';
import * as PostActions from 'actions/post_actions.jsx';

import Constants from 'utils/constants.jsx';

import {FormattedMessage} from 'react-intl';
import {browserHistory} from 'react-router/es6';

const ActionTypes = Constants.ActionTypes;
const KeyCodes = Constants.KeyCodes;

import {REACTION_PATTERN} from './create_post.jsx';

import React from 'react';

export default class CreateComment extends React.Component {
    constructor(props) {
        super(props);

        this.lastTime = 0;

        this.handleSubmit = this.handleSubmit.bind(this);
        this.handleSubmitPost = this.handleSubmitPost.bind(this);
        this.handleSubmitReaction = this.handleSubmitReaction.bind(this);
        this.commentMsgKeyPress = this.commentMsgKeyPress.bind(this);
        this.handleChange = this.handleChange.bind(this);
        this.handleKeyDown = this.handleKeyDown.bind(this);
        this.handleBlur = this.handleBlur.bind(this);
        this.handleUploadClick = this.handleUploadClick.bind(this);
        this.handleUploadStart = this.handleUploadStart.bind(this);
        this.handleFileUploadComplete = this.handleFileUploadComplete.bind(this);
        this.handleUploadError = this.handleUploadError.bind(this);
        this.removePreview = this.removePreview.bind(this);
        this.getFileCount = this.getFileCount.bind(this);
        this.onPreferenceChange = this.onPreferenceChange.bind(this);
        this.focusTextbox = this.focusTextbox.bind(this);
        this.showPostDeletedModal = this.showPostDeletedModal.bind(this);
        this.hidePostDeletedModal = this.hidePostDeletedModal.bind(this);
        this.handlePostError = this.handlePostError.bind(this);

        PostStore.clearCommentDraftUploads();
        MessageHistoryStore.resetHistoryIndex('comment');

        const draft = PostStore.getCommentDraft(this.props.rootId);
        this.state = {
            message: draft.message,
            uploadsInProgress: draft.uploadsInProgress,
            fileInfos: draft.fileInfos,
            submitting: false,
            ctrlSend: PreferenceStore.getBool(Constants.Preferences.CATEGORY_ADVANCED_SETTINGS, 'send_on_ctrl_enter'),
            showPostDeletedModal: false
        };

        this.lastBlurAt = 0;
    }

    componentDidMount() {
        PreferenceStore.addChangeListener(this.onPreferenceChange);
        this.focusTextbox();
    }

    componentWillUnmount() {
        PreferenceStore.removeChangeListener(this.onPreferenceChange);
    }

    onPreferenceChange() {
        this.setState({
            ctrlSend: PreferenceStore.getBool(Constants.Preferences.CATEGORY_ADVANCED_SETTINGS, 'send_on_ctrl_enter')
        });
    }

    componentDidUpdate(prevProps, prevState) {
        if (prevState.uploadsInProgress < this.state.uploadsInProgress) {
            $('.post-right__scroll').scrollTop($('.post-right__scroll')[0].scrollHeight);
        }

        if (prevProps.rootId !== this.props.rootId) {
            this.focusTextbox();
        }
    }

    handlePostError(postError) {
        this.setState({postError});
    }

    handleSubmit(e) {
        e.preventDefault();

        if (this.state.uploadsInProgress.length > 0) {
            return;
        }

        if (this.state.submitting) {
            return;
        }

        const message = this.state.message;

        if (this.state.postError) {
            this.setState({errorClass: 'animation--highlight'});
            setTimeout(() => {
                this.setState({errorClass: null});
            }, Constants.ANIMATION_TIMEOUT);
            return;
        }

        MessageHistoryStore.storeMessageInHistory(message);
        if (message.trim().length === 0 && this.state.fileInfos.length === 0) {
            return;
        }

        const isReaction = REACTION_PATTERN.exec(message);
        if (isReaction && EmojiStore.has(isReaction[2])) {
            this.handleSubmitReaction(isReaction);
        } else if (message.indexOf('/') === 0) {
            this.handleSubmitCommand(message);
        } else {
            this.handleSubmitPost(message);
        }

        this.setState({
            message: '',
            submitting: false,
            postError: null,
            fileInfos: [],
            serverError: null
        });

        const fasterThanHumanWillClick = 150;
        const forceFocus = (Date.now() - this.lastBlurAt < fasterThanHumanWillClick);
        this.focusTextbox(forceFocus);
    }

    handleSubmitCommand(message) {
        PostStore.storeCommentDraft(this.props.rootId, null);
        this.setState({message: '', postError: null, fileInfos: []});

        const args = {};
        args.channel_id = this.props.channelId;
        args.root_id = this.props.rootId;
        args.parent_id = this.props.rootId;
        ChannelActions.executeCommand(
            message,
            args,
            (data) => {
                this.setState({submitting: false});
                if (data.goto_location && data.goto_location.length > 0) {
                    browserHistory.push(data.goto_location);
                }
            },
            (err) => {
                if (err.sendMessage) {
                    this.handleSubmitPost(message);
                } else {
                    const state = {};
                    state.serverError = err.message;
                    state.submitting = false;
                    this.setState(state);
                }
            }
        );
    }

    handleSubmitPost(message) {
        const userId = UserStore.getCurrentId();
        const time = Utils.getTimestamp();

        const post = {};
        post.file_ids = [];
        post.message = message;
        post.channel_id = this.props.channelId;
        post.root_id = this.props.rootId;
        post.parent_id = this.props.rootId;
        post.file_ids = this.state.fileInfos.map((info) => info.id);
        post.pending_post_id = `${userId}:${time}`;
        post.user_id = userId;
        post.create_at = time;

        GlobalActions.emitUserCommentedEvent(post);

        PostActions.queuePost(post, false, null,
            (err) => {
                if (err.id === 'api.post.create_post.root_id.app_error') {
                    this.showPostDeletedModal();
                } else {
                    this.forceUpdate();
                }

                this.setState({
                    submitting: false
                });
            }
        );

        this.setState({
            message: '',
            submitting: false,
            postError: null,
            fileInfos: [],
            serverError: null
        });

        const fasterThanHumanWillClick = 150;
        const forceFocus = (Date.now() - this.state.lastBlurAt < fasterThanHumanWillClick);
        this.focusTextbox(forceFocus);
    }

    handleSubmitReaction(isReaction) {
        const action = isReaction[1];

        const emojiName = isReaction[2];
        const postId = this.props.latestPostId;

        if (action === '+') {
            PostActions.addReaction(this.props.channelId, postId, emojiName);
        } else if (action === '-') {
            PostActions.removeReaction(this.props.channelId, postId, emojiName);
        }

        PostStore.storeCommentDraft(this.props.rootId, null);
    }

    commentMsgKeyPress(e) {
        if (!UserAgent.isMobile() && ((this.state.ctrlSend && e.ctrlKey) || !this.state.ctrlSend)) {
            if (e.which === KeyCodes.ENTER && !e.shiftKey && !e.altKey) {
                e.preventDefault();
                ReactDOM.findDOMNode(this.refs.textbox).blur();
                this.handleSubmit(e);
            }
        }

        GlobalActions.emitLocalUserTypingEvent(this.props.channelId, this.props.rootId);
    }

    handleChange(e) {
        const message = e.target.value;

        const draft = PostStore.getCommentDraft(this.props.rootId);
        draft.message = message;
        PostStore.storeCommentDraft(this.props.rootId, draft);

        $('.post-right__scroll').parent().scrollTop($('.post-right__scroll')[0].scrollHeight);

        this.setState({message});
    }

    handleKeyDown(e) {
        if (this.state.ctrlSend && e.keyCode === KeyCodes.ENTER && e.ctrlKey === true) {
            this.commentMsgKeyPress(e);
            return;
        }

        if (!e.ctrlKey && !e.metaKey && !e.altKey && !e.shiftKey && e.keyCode === KeyCodes.UP && this.state.message === '') {
            e.preventDefault();

            const lastPost = PostStore.getCurrentUsersLatestPost(this.props.channelId, this.props.rootId);
            if (!lastPost) {
                return;
            }

            AppDispatcher.handleViewAction({
                type: ActionTypes.RECEIVED_EDIT_POST,
                refocusId: '#reply_textbox',
                title: Utils.localizeMessage('create_comment.commentTitle', 'Comment'),
                message: lastPost.message,
                postId: lastPost.id,
                channelId: lastPost.channel_id,
                comments: PostStore.getCommentCount(lastPost)
            });
        }

        if ((e.ctrlKey || e.metaKey) && !e.altKey && !e.shiftKey && (e.keyCode === Constants.KeyCodes.UP || e.keyCode === Constants.KeyCodes.DOWN)) {
            const lastMessage = MessageHistoryStore.nextMessageInHistory(e.keyCode, this.state.message, 'comment');
            if (lastMessage !== null) {
                e.preventDefault();
                this.setState({
                    message: lastMessage
                });
            }
        }
    }

    handleUploadClick() {
        this.focusTextbox();
    }

    handleUploadStart(clientIds) {
        const draft = PostStore.getCommentDraft(this.props.rootId);

        draft.uploadsInProgress = draft.uploadsInProgress.concat(clientIds);
        PostStore.storeCommentDraft(this.props.rootId, draft);

        this.setState({uploadsInProgress: draft.uploadsInProgress});

        // this is a bit redundant with the code that sets focus when the file input is clicked,
        // but this also resets the focus after a drag and drop
        this.focusTextbox();
    }

    handleFileUploadComplete(fileInfos, clientIds) {
        const draft = PostStore.getCommentDraft(this.props.rootId);

        // remove each finished file from uploads
        for (let i = 0; i < clientIds.length; i++) {
            const index = draft.uploadsInProgress.indexOf(clientIds[i]);

            if (index !== -1) {
                draft.uploadsInProgress.splice(index, 1);
            }
        }

        draft.fileInfos = draft.fileInfos.concat(fileInfos);
        PostStore.storeCommentDraft(this.props.rootId, draft);

        this.setState({uploadsInProgress: draft.uploadsInProgress, fileInfos: draft.fileInfos});
    }

    handleUploadError(err, clientId) {
        if (clientId === -1) {
            this.setState({serverError: err});
        } else {
            const draft = PostStore.getCommentDraft(this.props.rootId);

            const index = draft.uploadsInProgress.indexOf(clientId);
            if (index !== -1) {
                draft.uploadsInProgress.splice(index, 1);
            }

            PostStore.storeCommentDraft(this.props.rootId, draft);

            this.setState({uploadsInProgress: draft.uploadsInProgress, serverError: err});
        }
    }

    removePreview(id) {
        const fileInfos = this.state.fileInfos;
        const uploadsInProgress = this.state.uploadsInProgress;

        // id can either be the id of an uploaded file or the client id of an in progress upload
        let index = fileInfos.findIndex((info) => info.id === id);
        if (index === -1) {
            index = uploadsInProgress.indexOf(id);

            if (index !== -1) {
                uploadsInProgress.splice(index, 1);
                this.refs.fileUpload.getWrappedInstance().cancelUpload(id);
            }
        } else {
            fileInfos.splice(index, 1);
        }

        const draft = PostStore.getCommentDraft(this.props.rootId);
        draft.fileInfos = fileInfos;
        draft.uploadsInProgress = uploadsInProgress;
        PostStore.storeCommentDraft(this.props.rootId, draft);

        this.setState({fileInfos, uploadsInProgress});
    }

    componentWillReceiveProps(newProps) {
        if (newProps.rootId !== this.props.rootId) {
            const draft = PostStore.getCommentDraft(newProps.rootId);
            this.setState({message: draft.message, uploadsInProgress: draft.uploadsInProgress, fileInfos: draft.fileInfos});
        }
    }

    getFileCount() {
        return this.state.fileInfos.length + this.state.uploadsInProgress.length;
    }

    focusTextbox(keepFocus = false) {
        if (keepFocus || !Utils.isMobile()) {
            this.refs.textbox.focus();
        }
    }

    showPostDeletedModal() {
        this.setState({
            showPostDeletedModal: true
        });
    }

    hidePostDeletedModal() {
        this.setState({
            showPostDeletedModal: false
        });
    }

    handleBlur() {
        this.lastBlurAt = Date.now();
    }

    render() {
        let serverError = null;
        if (this.state.serverError) {
            serverError = (
                <div className='form-group has-error'>
                    <label className='control-label'>{this.state.serverError}</label>
                </div>
            );
        }

        let postError = null;
        if (this.state.postError) {
            const postErrorClass = 'post-error' + (this.state.errorClass ? (' ' + this.state.errorClass) : '');
            postError = <label className={postErrorClass}>{this.state.postError}</label>;
        }

        let preview = null;
        if (this.state.fileInfos.length > 0 || this.state.uploadsInProgress.length > 0) {
            preview = (
                <FilePreview
                    fileInfos={this.state.fileInfos}
                    onRemove={this.removePreview}
                    uploadsInProgress={this.state.uploadsInProgress}
                />
            );
        }

        let uploadsInProgressText = null;
        if (this.state.uploadsInProgress.length > 0) {
            uploadsInProgressText = (
                <span className='pull-right post-right-comments-upload-in-progress'>
                    {this.state.uploadsInProgress.length === 1 ? (
                        <FormattedMessage
                            id='create_comment.file'
                            defaultMessage='File uploading'
                        />
                    ) : (
                        <FormattedMessage
                            id='create_comment.files'
                            defaultMessage='Files uploading'
                        />
                    )}
                </span>
            );
        }

        return (
            <form onSubmit={this.handleSubmit}>
                <div className='post-create'>
                    <div
                        id={this.props.rootId}
                        className='post-create-body comment-create-body'
                    >
                        <div className='post-body__cell'>
                            <Textbox
                                onChange={this.handleChange}
                                onKeyPress={this.commentMsgKeyPress}
                                onKeyDown={this.handleKeyDown}
                                handlePostError={this.handlePostError}
                                value={this.state.message}
                                onBlur={this.handleBlur}
                                createMessage={Utils.localizeMessage('create_comment.addComment', 'Add a comment...')}
                                initialText=''
                                channelId={this.props.channelId}
                                id='reply_textbox'
                                ref='textbox'
                            />
                            <FileUpload
                                ref='fileUpload'
                                getFileCount={this.getFileCount}
                                onClick={this.handleUploadClick}
                                onUploadStart={this.handleUploadStart}
                                onFileUpload={this.handleFileUploadComplete}
                                onUploadError={this.handleUploadError}
                                postType='comment'
                                channelId={this.props.channelId}
                            />
                        </div>
                    </div>
                    <MsgTyping
                        channelId={this.props.channelId}
                        parentId={this.props.rootId}
                    />
                    <div className='post-create-footer'>
                        <input
                            type='button'
                            className='btn btn-primary comment-btn pull-right'
                            value={Utils.localizeMessage('create_comment.comment', 'Add Comment')}
                            onClick={this.handleSubmit}
                        />
                        {uploadsInProgressText}
                        {postError}
                        {preview}
                        {serverError}
                    </div>
                </div>
                <PostDeletedModal
                    show={this.state.showPostDeletedModal}
                    onHide={this.hidePostDeletedModal}
                />
            </form>
        );
    }
}

CreateComment.propTypes = {
    channelId: React.PropTypes.string.isRequired,
    rootId: React.PropTypes.string.isRequired,
    latestPostId: React.PropTypes.string
};
