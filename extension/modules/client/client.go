package clientModule

import (
	client "github.com/Mrs4s/MiraiGo/client"
)

import (
	"github.com/dop251/goja"
	"github.com/gogap/gojs-tool/gojs"
)

var (
	module = gojs.NewGojaModule("client")
)

func init() {
	module.Set(
		gojs.Objects{
			// Functions
			"GenRandomDevice":     client.GenRandomDevice,
			"NewClient":           client.NewClient,
			"NewClientMd5":        client.NewClientMd5,
			"NewUinFilterPrivate": client.NewUinFilterPrivate,

			// Var and consts
			"Administrator":     client.Administrator,
			"Emotion":           client.Emotion,
			"EmptyBytes":        client.EmptyBytes,
			"ErrAlreadyOnline":  client.ErrAlreadyOnline,
			"Legend":            client.Legend,
			"Member":            client.Member,
			"NeedCaptcha":       client.NeedCaptcha,
			"NumberRange":       client.NumberRange,
			"OtherLoginError":   client.OtherLoginError,
			"Owner":             client.Owner,
			"Performer":         client.Performer,
			"StrongNewbie":      client.StrongNewbie,
			"SystemDeviceInfo":  client.SystemDeviceInfo,
			"Talkative":         client.Talkative,
			"UnknownLoginError": client.UnknownLoginError,
			"UnsafeDeviceError": client.UnsafeDeviceError,

			// Types (value type)
			"ClientDisconnectedEvent":    func() client.ClientDisconnectedEvent { return client.ClientDisconnectedEvent{} },
			"CurrentTalkative":           func() client.CurrentTalkative { return client.CurrentTalkative{} },
			"DeviceInfo":                 func() client.DeviceInfo { return client.DeviceInfo{} },
			"DeviceInfoFile":             func() client.DeviceInfoFile { return client.DeviceInfoFile{} },
			"FriendInfo":                 func() client.FriendInfo { return client.FriendInfo{} },
			"FriendListResponse":         func() client.FriendListResponse { return client.FriendListResponse{} },
			"FriendMessageRecalledEvent": func() client.FriendMessageRecalledEvent { return client.FriendMessageRecalledEvent{} },
			"GroupHonorInfo":             func() client.GroupHonorInfo { return client.GroupHonorInfo{} },
			"GroupInfo":                  func() client.GroupInfo { return client.GroupInfo{} },
			"GroupInvitedRequest":        func() client.GroupInvitedRequest { return client.GroupInvitedRequest{} },
			"GroupLeaveEvent":            func() client.GroupLeaveEvent { return client.GroupLeaveEvent{} },
			"GroupMemberInfo":            func() client.GroupMemberInfo { return client.GroupMemberInfo{} },
			"GroupMessageRecalledEvent":  func() client.GroupMessageRecalledEvent { return client.GroupMessageRecalledEvent{} },
			"GroupMuteEvent":             func() client.GroupMuteEvent { return client.GroupMuteEvent{} },
			"HonorMemberInfo":            func() client.HonorMemberInfo { return client.HonorMemberInfo{} },
			"LogEvent":                   func() client.LogEvent { return client.LogEvent{} },
			"LoginResponse":              func() client.LoginResponse { return client.LoginResponse{} },
			"MemberJoinGroupEvent":       func() client.MemberJoinGroupEvent { return client.MemberJoinGroupEvent{} },
			"MemberLeaveGroupEvent":      func() client.MemberLeaveGroupEvent { return client.MemberLeaveGroupEvent{} },
			"MemberPermissionChangedEvent": func() client.MemberPermissionChangedEvent {
				return client.MemberPermissionChangedEvent{}
			},
			"NewFriendEvent":       func() client.NewFriendEvent { return client.NewFriendEvent{} },
			"NewFriendRequest":     func() client.NewFriendRequest { return client.NewFriendRequest{} },
			"QQClient":             func() client.QQClient { return client.QQClient{} },
			"UserJoinGroupRequest": func() client.UserJoinGroupRequest { return client.UserJoinGroupRequest{} },
			"Version":              func() client.Version { return client.Version{} },
			"VipInfo":              func() client.VipInfo { return client.VipInfo{} },

			// Types (pointer type)
			"NewClientDisconnectedEvent": func() *client.ClientDisconnectedEvent { return &client.ClientDisconnectedEvent{} },
			"NewCurrentTalkative":        func() *client.CurrentTalkative { return &client.CurrentTalkative{} },
			"NewDeviceInfo":              func() *client.DeviceInfo { return &client.DeviceInfo{} },
			"NewDeviceInfoFile":          func() *client.DeviceInfoFile { return &client.DeviceInfoFile{} },
			"NewFriendInfo":              func() *client.FriendInfo { return &client.FriendInfo{} },
			"NewFriendListResponse":      func() *client.FriendListResponse { return &client.FriendListResponse{} },
			"NewFriendMessageRecalledEvent": func() *client.FriendMessageRecalledEvent {
				return &client.FriendMessageRecalledEvent{}
			},
			"NewGroupHonorInfo":            func() *client.GroupHonorInfo { return &client.GroupHonorInfo{} },
			"NewGroupInfo":                 func() *client.GroupInfo { return &client.GroupInfo{} },
			"NewGroupInvitedRequest":       func() *client.GroupInvitedRequest { return &client.GroupInvitedRequest{} },
			"NewGroupLeaveEvent":           func() *client.GroupLeaveEvent { return &client.GroupLeaveEvent{} },
			"NewGroupMemberInfo":           func() *client.GroupMemberInfo { return &client.GroupMemberInfo{} },
			"NewGroupMessageRecalledEvent": func() *client.GroupMessageRecalledEvent { return &client.GroupMessageRecalledEvent{} },
			"NewGroupMuteEvent":            func() *client.GroupMuteEvent { return &client.GroupMuteEvent{} },
			"NewHonorMemberInfo":           func() *client.HonorMemberInfo { return &client.HonorMemberInfo{} },
			"NewLogEvent":                  func() *client.LogEvent { return &client.LogEvent{} },
			"NewLoginResponse":             func() *client.LoginResponse { return &client.LoginResponse{} },
			"NewMemberJoinGroupEvent":      func() *client.MemberJoinGroupEvent { return &client.MemberJoinGroupEvent{} },
			"NewMemberLeaveGroupEvent":     func() *client.MemberLeaveGroupEvent { return &client.MemberLeaveGroupEvent{} },
			"NewMemberPermissionChangedEvent": func() *client.MemberPermissionChangedEvent {
				return &client.MemberPermissionChangedEvent{}
			},
			"NewNewFriendEvent":       func() *client.NewFriendEvent { return &client.NewFriendEvent{} },
			"NewNewFriendRequest":     func() *client.NewFriendRequest { return &client.NewFriendRequest{} },
			"NewQQClient":             func() *client.QQClient { return &client.QQClient{} },
			"NewUserJoinGroupRequest": func() *client.UserJoinGroupRequest { return &client.UserJoinGroupRequest{} },
			"NewVersion":              func() *client.Version { return &client.Version{} },
			"NewVipInfo":              func() *client.VipInfo { return &client.VipInfo{} },
		},
	).Register()
}

func Enable(runtime *goja.Runtime) {
	module.Enable(runtime)
}
