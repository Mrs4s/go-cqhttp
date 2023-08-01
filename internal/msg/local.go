package msg

import (
	"io"

	"github.com/Mrs4s/MiraiGo/message"
)

// Poke 拍一拍
type Poke struct {
	Target int64
}

// Type 获取元素类型ID
func (e *Poke) Type() message.ElementType {
	// Make message.IMessageElement Happy
	return message.At
}

// LocalImage 本地图片
type LocalImage struct {
	Stream io.ReadSeeker
	File   string
	URL    string

	Flash    bool
	EffectID int32
}

// Type implements the message.IMessageElement.
func (e *LocalImage) Type() message.ElementType {
	return message.Image
}

// LocalVideo 本地视频
type LocalVideo struct {
	File  string
	Thumb io.ReadSeeker
}

// Type impl message.IMessageElement
func (e *LocalVideo) Type() message.ElementType {
	return message.Video
}
