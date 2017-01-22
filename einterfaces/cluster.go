// Copyright (c) 2015 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package einterfaces

import (
	"github.com/mattermost/platform/model"
)

type ClusterInterface interface {
	StartInterNodeCommunication()
	StopInterNodeCommunication()
	GetClusterInfos() []*model.ClusterInfo
	GetClusterStats() ([]*model.ClusterStats, *model.AppError)
	RemoveAllSessionsForUserId(userId string)
	InvalidateCacheForUser(userId string)
	InvalidateCacheForChannel(channelId string)
	InvalidateCacheForChannelPosts(channelId string)
	Publish(event *model.WebSocketEvent)
	UpdateStatus(status *model.Status)
	GetLogs() ([]string, *model.AppError)
	GetClusterId() string
	ConfigChanged(previousConfig *model.Config, newConfig *model.Config, sendToOtherServer bool) *model.AppError
	InvalidateAllCaches() *model.AppError
}

var theClusterInterface ClusterInterface

func RegisterClusterInterface(newInterface ClusterInterface) {
	theClusterInterface = newInterface
}

func GetClusterInterface() ClusterInterface {
	return theClusterInterface
}
