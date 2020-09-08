package messageModule

import (
	message "github.com/Mrs4s/MiraiGo/message"
)

import (
	"github.com/dop251/goja"
	"github.com/gogap/gojs-tool/gojs"
)

var (
	module = gojs.NewGojaModule("message")
)

func init() {
	module.Set(
		gojs.Objects{
			// Functions
			"AtAll":             message.AtAll,
			"EstimateLength":    message.EstimateLength,
			"NewAt":             message.NewAt,
			"NewFace":           message.NewFace,
			"NewGroupImage":     message.NewGroupImage,
			"NewImage":          message.NewImage,
			"NewLightApp":       message.NewLightApp,
			"NewReply":          message.NewReply,
			"NewRichJson":       message.NewRichJson,
			"NewRichXml":        message.NewRichXml,
			"NewSendingMessage": message.NewSendingMessage,
			"NewText":           message.NewText,
			"NewUrlShare":       message.NewUrlShare,
			"ParseMessageElems": message.ParseMessageElems,
			"ToProtoElems":      message.ToProtoElems,
			"ToReadableString":  message.ToReadableString,
			"ToSrcProtoElems":   message.ToSrcProtoElems,

			// Var and consts
			"At":       message.At,
			"Face":     message.Face,
			"File":     message.File,
			"Forward":  message.Forward,
			"Image":    message.Image,
			"LightApp": message.LightApp,
			"Reply":    message.Reply,
			"Service":  message.Service,
			"Text":     message.Text,
			"Video":    message.Video,
			"Voice":    message.Voice,

			// Types (value type)
			"AtElement":          func() message.AtElement { return message.AtElement{} },
			"FaceElement":        func() message.FaceElement { return message.FaceElement{} },
			"ForwardElement":     func() message.ForwardElement { return message.ForwardElement{} },
			"ForwardMessage":     func() message.ForwardMessage { return message.ForwardMessage{} },
			"ForwardNode":        func() message.ForwardNode { return message.ForwardNode{} },
			"FriendImageElement": func() message.FriendImageElement { return message.FriendImageElement{} },
			"GroupFileElement":   func() message.GroupFileElement { return message.GroupFileElement{} },
			"GroupImageElement":  func() message.GroupImageElement { return message.GroupImageElement{} },
			"GroupMessage":       func() message.GroupMessage { return message.GroupMessage{} },
			"GroupVoiceElement":  func() message.GroupVoiceElement { return message.GroupVoiceElement{} },
			"ImageElement":       func() message.ImageElement { return message.ImageElement{} },
			"LightAppElement":    func() message.LightAppElement { return message.LightAppElement{} },
			"PrivateMessage":     func() message.PrivateMessage { return message.PrivateMessage{} },
			"ReplyElement":       func() message.ReplyElement { return message.ReplyElement{} },
			"Sender":             func() message.Sender { return message.Sender{} },
			"SendingMessage":     func() message.SendingMessage { return message.SendingMessage{} },
			"ServiceElement":     func() message.ServiceElement { return message.ServiceElement{} },
			"ShortVideoElement":  func() message.ShortVideoElement { return message.ShortVideoElement{} },
			"TempMessage":        func() message.TempMessage { return message.TempMessage{} },
			"TextElement":        func() message.TextElement { return message.TextElement{} },
			"VoiceElement":       func() message.VoiceElement { return message.VoiceElement{} },

			// Types (pointer type)
			"NewAtElement":          func() *message.AtElement { return &message.AtElement{} },
			"NewFaceElement":        func() *message.FaceElement { return &message.FaceElement{} },
			"NewForwardElement":     func() *message.ForwardElement { return &message.ForwardElement{} },
			"NewForwardMessage":     func() *message.ForwardMessage { return &message.ForwardMessage{} },
			"NewForwardNode":        func() *message.ForwardNode { return &message.ForwardNode{} },
			"NewFriendImageElement": func() *message.FriendImageElement { return &message.FriendImageElement{} },
			"NewGroupFileElement":   func() *message.GroupFileElement { return &message.GroupFileElement{} },
			"NewGroupImageElement":  func() *message.GroupImageElement { return &message.GroupImageElement{} },
			"NewGroupMessage":       func() *message.GroupMessage { return &message.GroupMessage{} },
			"NewGroupVoiceElement":  func() *message.GroupVoiceElement { return &message.GroupVoiceElement{} },
			"NewImageElement":       func() *message.ImageElement { return &message.ImageElement{} },
			"NewLightAppElement":    func() *message.LightAppElement { return &message.LightAppElement{} },
			"NewPrivateMessage":     func() *message.PrivateMessage { return &message.PrivateMessage{} },
			"NewReplyElement":       func() *message.ReplyElement { return &message.ReplyElement{} },
			"NewSender":             func() *message.Sender { return &message.Sender{} },
			"NewServiceElement":     func() *message.ServiceElement { return &message.ServiceElement{} },
			"NewShortVideoElement":  func() *message.ShortVideoElement { return &message.ShortVideoElement{} },
			"NewTempMessage":        func() *message.TempMessage { return &message.TempMessage{} },
			"NewTextElement":        func() *message.TextElement { return &message.TextElement{} },
			"NewVoiceElement":       func() *message.VoiceElement { return &message.VoiceElement{} },
		},
	).Register()
}

func Enable(runtime *goja.Runtime) {
	module.Enable(runtime)
}
